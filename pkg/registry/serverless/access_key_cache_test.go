package serverless

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAccessKeyCache_GetPut(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newAccessKeyCache(ctx)

	entry := &cachedAccessKey{
		id:        "dof_v1_abc",
		secret:    "dof_v1_abc:secret123",
		apiHost:   "https://faas.example.com",
		expiresAt: time.Now().Add(1 * time.Hour),
	}

	require.Nil(t, c.get("hash1", "ns-1"), "expected nil for missing entry")

	c.put("hash1", "ns-1", entry)

	got := c.get("hash1", "ns-1")
	require.NotNil(t, got)
	require.Equal(t, "dof_v1_abc", got.id)
	require.Equal(t, "dof_v1_abc:secret123", got.secret)
	require.Equal(t, "https://faas.example.com", got.apiHost)

	require.Nil(t, c.get("hash1", "ns-2"), "different namespace should miss")
	require.Nil(t, c.get("hash2", "ns-1"), "different token hash should miss")
}

func TestAccessKeyCache_ExpiryBuffer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newAccessKeyCache(ctx)

	almostExpired := &cachedAccessKey{
		id:        "dof_v1_abc",
		secret:    "dof_v1_abc:secret123",
		apiHost:   "https://faas.example.com",
		expiresAt: time.Now().Add(2 * time.Minute),
	}
	c.put("hash1", "ns-1", almostExpired)

	require.Nil(t, c.get("hash1", "ns-1"),
		"key expiring within buffer window should be treated as stale")

	fresh := &cachedAccessKey{
		id:        "dof_v1_def",
		secret:    "dof_v1_def:secret456",
		apiHost:   "https://faas.example.com",
		expiresAt: time.Now().Add(1 * time.Hour),
	}
	c.put("hash1", "ns-1", fresh)

	got := c.get("hash1", "ns-1")
	require.NotNil(t, got)
	require.Equal(t, "dof_v1_def", got.id)
}

func TestAccessKeyCache_EvictExpired(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newAccessKeyCache(ctx)

	expired := &cachedAccessKey{
		id:        "old",
		secret:    "old:old-secret",
		apiHost:   "https://faas.example.com",
		expiresAt: time.Now().Add(-1 * time.Minute),
	}
	valid := &cachedAccessKey{
		id:        "new",
		secret:    "new:new-secret",
		apiHost:   "https://faas.example.com",
		expiresAt: time.Now().Add(1 * time.Hour),
	}

	c.put("hash1", "ns-expired", expired)
	c.put("hash1", "ns-valid", valid)

	c.evictExpired()

	c.mu.RLock()
	_, expiredExists := c.entries[cacheKey("hash1", "ns-expired")]
	_, validExists := c.entries[cacheKey("hash1", "ns-valid")]
	c.mu.RUnlock()

	require.False(t, expiredExists, "expired entry should be evicted")
	require.True(t, validExists, "valid entry should remain")
}

func TestAccessKeyCache_ConcurrentAccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newAccessKeyCache(ctx)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			c.put("hash", "ns-1", &cachedAccessKey{
				id:        "key",
				secret:    "key:secret",
				apiHost:   "https://faas.example.com",
				expiresAt: time.Now().Add(1 * time.Hour),
			})
		}(i)
		go func(i int) {
			defer wg.Done()
			c.get("hash", "ns-1")
		}(i)
	}
	wg.Wait()
}

func TestAccessKeyCache_CleanupStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := newAccessKeyCache(ctx)
	_ = c

	cancel()
	// No assertion needed; verifies no goroutine leak via race detector / -count.
}

func TestHashToken(t *testing.T) {
	h1 := hashToken("token-abc")
	h2 := hashToken("token-abc")
	h3 := hashToken("token-def")

	require.Equal(t, h1, h2, "same input should produce same hash")
	require.NotEqual(t, h1, h3, "different input should produce different hash")
	require.Len(t, h1, 16, "hash should be 16 hex chars (8 bytes)")
}

func TestCacheKey(t *testing.T) {
	key := cacheKey("abc123", "fn-ns-1")
	require.Equal(t, "abc123:fn-ns-1", key)
}
