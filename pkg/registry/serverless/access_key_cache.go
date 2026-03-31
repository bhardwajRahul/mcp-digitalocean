package serverless

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

const (
	accessKeyTTL          = "1d"
	accessKeyExpiryBuffer = 5 * time.Minute
	cacheCleanupInterval  = 10 * time.Minute
	accessKeyClientName   = "mcp-server"
)

// AccessKeyCreateRequest represents a request to create a namespace access key.
// Fields match godo.FunctionsAccessKeyCreateRequest.
type AccessKeyCreateRequest struct {
	Name      string
	ExpiresIn string // duration string, e.g. "2h"
}

// AccessKey represents a namespace access key for OpenWhisk API authentication.
// ID and Secret map to godo.FunctionsAccessKey fields; APIHost is populated
// from the namespace since godo's access key response does not include it.
type AccessKey struct {
	ID        string // godo field "id" -- username for basic auth
	Secret    string // godo field "secret" -- full "<id>:<secret>" credential
	APIHost   string // from FunctionsNamespace.ApiHost, not the access key itself
	ExpiresAt time.Time
	Name      string
}

// AccessKeyService creates and resolves namespace access keys.
// The implementation is responsible for fetching the namespace API host
// (since godo's FunctionsAccessKey does not include it) and combining it
// with the access key credentials.
type AccessKeyService interface {
	CreateAccessKey(ctx context.Context, namespace string, req *AccessKeyCreateRequest) (*AccessKey, error)
}

type cachedAccessKey struct {
	id        string
	secret    string
	apiHost   string
	expiresAt time.Time
}

func (c *cachedAccessKey) isExpiringSoon(buffer time.Duration) bool {
	return time.Now().Add(buffer).After(c.expiresAt)
}

type accessKeyCache struct {
	mu      sync.RWMutex
	entries map[string]*cachedAccessKey
}

func newAccessKeyCache(ctx context.Context) *accessKeyCache {
	c := &accessKeyCache{
		entries: make(map[string]*cachedAccessKey),
	}
	c.startCleanup(ctx)
	return c
}

func (c *accessKeyCache) get(tokenHash, namespaceID string) *cachedAccessKey {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[cacheKey(tokenHash, namespaceID)]
	if !ok {
		return nil
	}
	if entry.isExpiringSoon(accessKeyExpiryBuffer) {
		return nil
	}
	return entry
}

func (c *accessKeyCache) put(tokenHash, namespaceID string, ak *cachedAccessKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[cacheKey(tokenHash, namespaceID)] = ak
}

func (c *accessKeyCache) evictExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, v := range c.entries {
		if now.After(v.expiresAt) {
			delete(c.entries, k)
		}
	}
}

func (c *accessKeyCache) startCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(cacheCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.evictExpired()
			}
		}
	}()
}

func cacheKey(tokenHash, namespaceID string) string {
	return tokenHash + ":" + namespaceID
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:8])
}
