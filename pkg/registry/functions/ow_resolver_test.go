package functions

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	middleware "mcp-digitalocean/internal"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)


func mockClient(mock *MockFunctionsService) func(ctx context.Context) (*godo.Client, error) {
	return func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Functions: mock}, nil
	}
}

func testNS() *godo.FunctionsNamespace {
	return &godo.FunctionsNamespace{
		Namespace: "ns-name-123",
		ApiHost:   "https://faas.example.com",
		Label:     "test",
		Region:    "nyc1",
	}
}

func setupResolverTest(t *testing.T) (*OWResolver, *MockFunctionsService) {
	ctrl := gomock.NewController(t)
	mock := NewMockFunctionsService(ctrl)
	resolver := NewOWResolver(mockClient(mock))
	return resolver, mock
}

func expectFullResolve(mock *MockFunctionsService, nsID string) {
	mock.EXPECT().GetNamespace(gomock.Any(), nsID).
		Return(testNS(), nil, nil)
	mock.EXPECT().ListAccessKeys(gomock.Any(), nsID).
		Return([]godo.FunctionsAccessKey{}, nil, nil)
	mock.EXPECT().CreateAccessKey(gomock.Any(), nsID, gomock.Any()).
		Return(&godo.FunctionsAccessKey{
			ID:     "key-id-1",
			Secret: "key-secret-1",
			Name:   "mcp-do-12345",
		}, nil, nil)
}

func TestOWResolver_CacheMiss(t *testing.T) {
	resolver, mock := setupResolverTest(t)
	expectFullResolve(mock, "ns-uuid-1")

	ow, nsName, err := resolver.Resolve(context.Background(), "ns-uuid-1")
	require.NoError(t, err)
	require.NotNil(t, ow)
	require.Equal(t, "ns-name-123", nsName)
	require.Equal(t, "https://faas.example.com", ow.apiHost)
	require.Contains(t, ow.authKey, "key-id-1:")
}

func TestOWResolver_CacheHit(t *testing.T) {
	resolver, mock := setupResolverTest(t)
	expectFullResolve(mock, "ns-uuid-1")

	// First call populates the cache.
	_, _, err := resolver.Resolve(context.Background(), "ns-uuid-1")
	require.NoError(t, err)

	// Second call should hit cache — no more godo calls expected.
	ow, nsName, err := resolver.Resolve(context.Background(), "ns-uuid-1")
	require.NoError(t, err)
	require.NotNil(t, ow)
	require.Equal(t, "ns-name-123", nsName)
}

func TestOWResolver_ExpiredEntry(t *testing.T) {
	resolver, mock := setupResolverTest(t)

	mock.EXPECT().GetNamespace(gomock.Any(), "ns-uuid-1").
		Return(testNS(), nil, nil).Times(2)
	mock.EXPECT().ListAccessKeys(gomock.Any(), "ns-uuid-1").
		Return([]godo.FunctionsAccessKey{}, nil, nil).Times(2)
	mock.EXPECT().CreateAccessKey(gomock.Any(), "ns-uuid-1", gomock.Any()).
		Return(&godo.FunctionsAccessKey{
			ID:     "key-1",
			Secret: "secret-1",
			Name:   "mcp-do-1",
		}, nil, nil).Times(2)

	// First resolve.
	_, _, err := resolver.Resolve(context.Background(), "ns-uuid-1")
	require.NoError(t, err)

	// Manually expire the cached entry.
	resolver.mu.Lock()
	for _, elem := range resolver.items {
		entry := elem.Value.(*lruEntry)
		entry.auth.validUntil = time.Now().Add(-1 * time.Minute)
	}
	resolver.mu.Unlock()

	// Second resolve should miss cache and call godo again.
	_, _, err = resolver.Resolve(context.Background(), "ns-uuid-1")
	require.NoError(t, err)
}

func TestOWResolver_PerUserCacheIsolation(t *testing.T) {
	resolver, mock := setupResolverTest(t)

	// Expect two full resolves — one per user.
	mock.EXPECT().GetNamespace(gomock.Any(), "ns-uuid-1").
		Return(testNS(), nil, nil).Times(2)
	mock.EXPECT().ListAccessKeys(gomock.Any(), "ns-uuid-1").
		Return([]godo.FunctionsAccessKey{}, nil, nil).Times(2)
	mock.EXPECT().CreateAccessKey(gomock.Any(), "ns-uuid-1", gomock.Any()).
		Return(&godo.FunctionsAccessKey{
			ID:     "key-1",
			Secret: "secret-1",
			Name:   "mcp-do-1",
		}, nil, nil).Times(2)

	ctxA := context.WithValue(context.Background(), middleware.AuthKey{}, "Bearer token-user-a")
	ctxB := context.WithValue(context.Background(), middleware.AuthKey{}, "Bearer token-user-b")

	_, _, err := resolver.Resolve(ctxA, "ns-uuid-1")
	require.NoError(t, err)
	_, _, err = resolver.Resolve(ctxB, "ns-uuid-1")
	require.NoError(t, err)

	// Each user should have their own cache entry.
	resolver.mu.Lock()
	require.Equal(t, 2, len(resolver.items))
	resolver.mu.Unlock()
}

func TestOWResolver_LRUEviction(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockFunctionsService(ctrl)
	resolver := NewOWResolver(mockClient(mock))

	mock.EXPECT().GetNamespace(gomock.Any(), gomock.Any()).
		Return(testNS(), nil, nil).AnyTimes()
	mock.EXPECT().ListAccessKeys(gomock.Any(), gomock.Any()).
		Return([]godo.FunctionsAccessKey{}, nil, nil).AnyTimes()
	mock.EXPECT().CreateAccessKey(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, req *godo.FunctionsAccessKeyCreateRequest) (*godo.FunctionsAccessKey, *godo.Response, error) {
			return &godo.FunctionsAccessKey{
				ID:     "key-" + req.Name,
				Secret: "secret",
				Name:   req.Name,
			}, nil, nil
		}).AnyTimes()

	// Fill beyond maxCacheEntries.
	for i := 0; i < maxCacheEntries+10; i++ {
		nsID := "ns-" + strings.Repeat("x", 5) + "-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		_, _, err := resolver.Resolve(context.Background(), nsID)
		require.NoError(t, err)
	}

	resolver.mu.Lock()
	require.LessOrEqual(t, len(resolver.items), maxCacheEntries)
	resolver.mu.Unlock()
}

func TestOWResolver_OrphanCleanup(t *testing.T) {
	resolver, mock := setupResolverTest(t)

	orphanKeys := []godo.FunctionsAccessKey{
		{ID: "orphan-1", Name: "mcp-do-old-1"},
		{ID: "orphan-2", Name: "mcp-do-old-2"},
		{ID: "user-key", Name: "my-custom-key"},
	}

	mock.EXPECT().GetNamespace(gomock.Any(), "ns-uuid-1").
		Return(testNS(), nil, nil)
	mock.EXPECT().ListAccessKeys(gomock.Any(), "ns-uuid-1").
		Return(orphanKeys, nil, nil)
	// Only the mcp-do- prefixed keys should be deleted.
	mock.EXPECT().DeleteAccessKey(gomock.Any(), "ns-uuid-1", "orphan-1").Return(nil, nil)
	mock.EXPECT().DeleteAccessKey(gomock.Any(), "ns-uuid-1", "orphan-2").Return(nil, nil)
	mock.EXPECT().CreateAccessKey(gomock.Any(), "ns-uuid-1", gomock.Any()).
		Return(&godo.FunctionsAccessKey{
			ID: "new-key", Secret: "new-secret", Name: "mcp-do-new",
		}, nil, nil)

	_, _, err := resolver.Resolve(context.Background(), "ns-uuid-1")
	require.NoError(t, err)
}

func TestOWResolver_GetNamespaceError(t *testing.T) {
	resolver, mock := setupResolverTest(t)

	mock.EXPECT().GetNamespace(gomock.Any(), "bad-ns").
		Return(nil, nil, &godo.ErrorResponse{Message: "not found"})

	_, _, err := resolver.Resolve(context.Background(), "bad-ns")
	require.Error(t, err)
	require.Contains(t, err.Error(), "get namespace")
}

func TestOWResolver_MissingApiHost(t *testing.T) {
	resolver, mock := setupResolverTest(t)

	mock.EXPECT().GetNamespace(gomock.Any(), "ns-no-host").
		Return(&godo.FunctionsNamespace{Namespace: "ns", ApiHost: ""}, nil, nil)

	_, _, err := resolver.Resolve(context.Background(), "ns-no-host")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no api_host")
}

func TestOWResolver_CreateKeyError(t *testing.T) {
	resolver, mock := setupResolverTest(t)

	mock.EXPECT().GetNamespace(gomock.Any(), "ns-uuid-1").
		Return(testNS(), nil, nil)
	mock.EXPECT().ListAccessKeys(gomock.Any(), "ns-uuid-1").
		Return([]godo.FunctionsAccessKey{}, nil, nil)
	mock.EXPECT().CreateAccessKey(gomock.Any(), "ns-uuid-1", gomock.Any()).
		Return(nil, nil, &godo.ErrorResponse{Message: "forbidden"})

	_, _, err := resolver.Resolve(context.Background(), "ns-uuid-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "create access key")
}

func TestOWResolver_ConcurrentAccess(t *testing.T) {
	resolver, mock := setupResolverTest(t)

	mock.EXPECT().GetNamespace(gomock.Any(), gomock.Any()).
		Return(testNS(), nil, nil).AnyTimes()
	mock.EXPECT().ListAccessKeys(gomock.Any(), gomock.Any()).
		Return([]godo.FunctionsAccessKey{}, nil, nil).AnyTimes()
	mock.EXPECT().CreateAccessKey(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&godo.FunctionsAccessKey{
			ID: "key-conc", Secret: "secret-conc", Name: "mcp-do-conc",
		}, nil, nil).AnyTimes()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _, err := resolver.Resolve(context.Background(), "ns-uuid-1")
			require.NoError(t, err)
		}(i)
	}
	wg.Wait()
}

func TestCacheKey_StdioMode(t *testing.T) {
	// No auth on context → key should just be the namespace ID.
	key := cacheKey(context.Background(), "ns-123")
	require.Equal(t, "ns-123", key)
}

func TestCacheKey_HTTPMode(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.AuthKey{}, "Bearer token-abc")
	key := cacheKey(ctx, "ns-123")
	require.Contains(t, key, ":ns-123")
	require.NotEqual(t, "ns-123", key)
}

func TestCacheKey_DifferentTokens(t *testing.T) {
	ctxA := context.WithValue(context.Background(), middleware.AuthKey{}, "Bearer token-a")
	ctxB := context.WithValue(context.Background(), middleware.AuthKey{}, "Bearer token-b")

	keyA := cacheKey(ctxA, "ns-123")
	keyB := cacheKey(ctxB, "ns-123")
	require.NotEqual(t, keyA, keyB)
}
