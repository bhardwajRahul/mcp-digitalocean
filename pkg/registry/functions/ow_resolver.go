package functions

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	middleware "mcp-digitalocean/internal"

	"github.com/digitalocean/godo"
	"golang.org/x/sync/singleflight"
)

const (
	mcpKeyPrefix = "mcp-do-"
	keyTTL       = "24h"
	// Refresh the cached key when it has less than this duration remaining,
	// so we never hand out a key that expires mid-request.
	keyRefreshBuffer = time.Hour
	// Maximum entries in the auth cache. Prevents unbounded memory growth
	// under high-cardinality workloads (many users × namespaces). Sized to
	// comfortably absorb demo-driven traffic spikes (~6000 concurrent
	// user/namespace pairs, ~3 MB of memory at full capacity) while still
	// evicting stale credentials for users who haven't returned recently.
	maxCacheEntries = 6000
)

// cachedAuth holds a resolved OW client and its expiry metadata.
type cachedAuth struct {
	ow         *owClient
	nsName     string
	validUntil time.Time
}

// lruEntry pairs a cachedAuth with its map key so the LRU list element can
// remove the corresponding map entry in O(1).
type lruEntry struct {
	key  string
	auth *cachedAuth
}

// OWResolver lazily provisions short-lived access keys for the OpenWhisk data
// plane and caches them in a bounded LRU cache with TTL expiration. On first
// use for a given namespace it:
//  1. Fetches namespace metadata (api_host, namespace name).
//  2. Lists existing access keys and deletes any with the "mcp-do-" prefix
//     (orphans from previous sessions).
//  3. Creates a new 24h access key.
//  4. Caches the result until near-expiry, evicting least-recently-used entries
//     when the cache is full.
type OWResolver struct {
	client func(ctx context.Context) (*godo.Client, error)
	mu     sync.Mutex
	items  map[string]*list.Element // cache key → list element
	order  *list.List               // front = most recently used
	flight singleflight.Group       // deduplicates concurrent resolves for the same key
}

func NewOWResolver(client func(ctx context.Context) (*godo.Client, error)) *OWResolver {
	return &OWResolver{
		client: client,
		items:  make(map[string]*list.Element),
		order:  list.New(),
	}
}

// cacheKey derives a cache key that is scoped to both the caller's identity
// and the namespace. In HTTP transport mode each user has a distinct auth token
// on the context, so different users get separate cache entries. In stdio mode
// (no per-request auth) we fall back to namespace-only keying.
func cacheKey(ctx context.Context, namespaceID string) string {
	auth, _ := ctx.Value(middleware.AuthKey{}).(string)
	if auth == "" {
		return namespaceID
	}
	h := sha256.Sum256([]byte(auth))
	return hex.EncodeToString(h[:8]) + ":" + namespaceID
}

// resolveResult is the value type shared via singleflight so concurrent
// callers for the same cache key receive the same result.
type resolveResult struct {
	ow     *owClient
	nsName string
}

// Resolve returns an authenticated OW client and the OW namespace name for the
// given DO namespace UUID. It creates or reuses a cached access key scoped to
// the caller's identity. Concurrent requests for the same cache key are
// deduplicated via singleflight so only one access key is created.
func (r *OWResolver) Resolve(ctx context.Context, namespaceID string) (*owClient, string, error) {
	ck := cacheKey(ctx, namespaceID)

	r.mu.Lock()
	if elem, ok := r.items[ck]; ok {
		entry := elem.Value.(*lruEntry)
		if time.Now().Before(entry.auth.validUntil) {
			r.order.MoveToFront(elem)
			r.mu.Unlock()
			return entry.auth.ow, entry.auth.nsName, nil
		}
		r.order.Remove(elem)
		delete(r.items, ck)
	}
	r.mu.Unlock()

	v, err, _ := r.flight.Do(ck, func() (interface{}, error) {
		return r.resolveAndCache(ctx, ck, namespaceID)
	})
	if err != nil {
		return nil, "", err
	}
	res := v.(*resolveResult)
	return res.ow, res.nsName, nil
}

// resolveAndCache fetches namespace metadata, cleans up orphaned keys, creates
// a new access key, and caches the result. Called at most once per cache key
// at any given time thanks to singleflight.
func (r *OWResolver) resolveAndCache(ctx context.Context, ck, namespaceID string) (*resolveResult, error) {
	gc, err := r.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	// This call acts as the authorization check: if the caller's token cannot
	// access this namespace, godo returns an error and we never reach key creation.
	ns, _, err := gc.Functions.GetNamespace(ctx, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("get namespace: %w", err)
	}
	if ns.ApiHost == "" {
		return nil, fmt.Errorf("namespace %s has no api_host", namespaceID)
	}

	r.cleanupOrphanedKeys(ctx, gc, namespaceID)

	key, err := r.createKey(ctx, gc, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("create access key for namespace %s: %w (function:admin permission required)", namespaceID, err)
	}

	authKey := key.ID + ":" + key.Secret
	ow := newOWClient(ns.ApiHost, authKey)

	entry := &cachedAuth{
		ow:         ow,
		nsName:     ns.Namespace,
		validUntil: time.Now().Add(24*time.Hour - keyRefreshBuffer),
	}

	r.mu.Lock()
	r.putLocked(ck, entry)
	r.mu.Unlock()

	return &resolveResult{ow: ow, nsName: ns.Namespace}, nil
}

// putLocked inserts or replaces a cache entry, evicts expired entries, and
// removes the least-recently-used entry if the cache is at capacity.
// Must be called with r.mu held.
func (r *OWResolver) putLocked(key string, auth *cachedAuth) {
	if elem, ok := r.items[key]; ok {
		r.order.Remove(elem)
		delete(r.items, key)
	}

	r.sweepExpiredLocked()

	for r.order.Len() >= maxCacheEntries {
		back := r.order.Back()
		if back == nil {
			break
		}
		e := back.Value.(*lruEntry)
		r.order.Remove(back)
		delete(r.items, e.key)
	}

	elem := r.order.PushFront(&lruEntry{key: key, auth: auth})
	r.items[key] = elem
}

// sweepExpiredLocked removes all entries whose validUntil has passed.
// Must be called with r.mu held.
func (r *OWResolver) sweepExpiredLocked() {
	now := time.Now()
	for elem := r.order.Back(); elem != nil; {
		prev := elem.Prev()
		if now.After(elem.Value.(*lruEntry).auth.validUntil) {
			e := elem.Value.(*lruEntry)
			r.order.Remove(elem)
			delete(r.items, e.key)
		}
		elem = prev
	}
}

func (r *OWResolver) cleanupOrphanedKeys(ctx context.Context, gc *godo.Client, namespaceID string) {
	keys, _, err := gc.Functions.ListAccessKeys(ctx, namespaceID)
	if err != nil {
		return
	}

	for _, k := range keys {
		if strings.HasPrefix(k.Name, mcpKeyPrefix) {
			gc.Functions.DeleteAccessKey(ctx, namespaceID, k.ID) //nolint:errcheck
		}
	}
}

func (r *OWResolver) createKey(ctx context.Context, gc *godo.Client, namespaceID string) (*godo.FunctionsAccessKey, error) {
	name := mcpKeyPrefix + fmt.Sprintf("%d", time.Now().UnixMilli())
	key, _, err := gc.Functions.CreateAccessKey(ctx, namespaceID, &godo.FunctionsAccessKeyCreateRequest{
		Name:      name,
		ExpiresIn: keyTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("create key API call: %w", err)
	}
	if key == nil {
		return nil, fmt.Errorf("empty response from create key")
	}

	return key, nil
}
