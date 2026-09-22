package xgoesi_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gohugoio/httpcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

// memoryCache is a minimal in-memory implementation of httpcache.Cache for tests.
type memoryCache struct {
	mu    sync.Mutex
	items map[string][]byte
}

func newMemoryCache() *memoryCache {
	return &memoryCache{items: make(map[string][]byte)}
}

func (c *memoryCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.items[key]
	return v, ok
}

func (c *memoryCache) Set(key string, v []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = v
}

func (c *memoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func TestResponseFromCache(t *testing.T) {
	t.Run("should report false for a nil response", func(t *testing.T) {
		assert.False(t, xgoesi.ResponseFromCache(nil))
	})
	t.Run("should report false when the header is absent", func(t *testing.T) {
		resp := &http.Response{Header: make(http.Header)}
		assert.False(t, xgoesi.ResponseFromCache(resp))
	})
	t.Run("should report true when the header is set", func(t *testing.T) {
		resp := &http.Response{Header: make(http.Header)}
		resp.Header.Set(httpcache.XFromCache, "1")
		assert.True(t, xgoesi.ResponseFromCache(resp))
	})
}

func TestForceRefreshContext(t *testing.T) {
	t.Run("should report false for a plain context", func(t *testing.T) {
		assert.False(t, xgoesi.IsForceRefresh(context.Background()))
	})
	t.Run("should report true for a marked context", func(t *testing.T) {
		ctx := xgoesi.NewContextWithForceRefresh(context.Background())
		assert.True(t, xgoesi.IsForceRefresh(ctx))
	})
}

func TestCacheKeyWithForceRefresh(t *testing.T) {
	newRequest := func(t *testing.T, ctx context.Context, method string) *http.Request {
		req, err := http.NewRequestWithContext(ctx, method, "https://esi.evetech.net/characters/42/skillqueue", nil)
		require.NoError(t, err)
		return req
	}
	t.Run("should return empty string when context is marked for force refresh", func(t *testing.T) {
		ctx := xgoesi.NewContextWithForceRefresh(context.Background())
		req := newRequest(t, ctx, http.MethodGet)
		assert.Equal(t, "", xgoesi.CacheKeyWithForceRefresh(req))
	})
	t.Run("should return the URL for a normal GET request", func(t *testing.T) {
		req := newRequest(t, context.Background(), http.MethodGet)
		assert.Equal(t, req.URL.String(), xgoesi.CacheKeyWithForceRefresh(req))
	})
	t.Run("should return empty string for a normal POST request", func(t *testing.T) {
		req := newRequest(t, context.Background(), http.MethodPost)
		assert.Equal(t, "", xgoesi.CacheKeyWithForceRefresh(req))
	})
	t.Run("should return the URL for a normal HEAD request", func(t *testing.T) {
		req := newRequest(t, context.Background(), http.MethodHead)
		assert.Equal(t, req.URL.String(), xgoesi.CacheKeyWithForceRefresh(req))
	})
	t.Run("should return empty string for a ranged request", func(t *testing.T) {
		req := newRequest(t, context.Background(), http.MethodGet)
		req.Header.Set("Range", "bytes=0-100")
		assert.Equal(t, "", xgoesi.CacheKeyWithForceRefresh(req))
	})
	t.Run("should integrate with httpcache to force a real network fetch", func(t *testing.T) {
		var calls int
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Cache-Control", "max-age=3600")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("response"))
		})
		ts := httptest.NewServer(mockHandler)
		defer ts.Close()

		transport := &httpcache.Transport{
			Cache:    newMemoryCache(),
			CacheKey: xgoesi.CacheKeyWithForceRefresh,
		}
		client := &http.Client{Transport: transport}
		do := func(ctx context.Context) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
			require.NoError(t, err)
			resp, err := client.Do(req)
			require.NoError(t, err)
			// httpcache only writes to the cache once the body reaches EOF.
			_, err = io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
		}

		// first request populates the cache
		do(context.Background())
		assert.Equal(t, 1, calls)

		// second, normal request is served from cache without a network call
		do(context.Background())
		assert.Equal(t, 1, calls, "expected cached response to be used")

		// a force-refreshed request bypasses the cache and hits the network again
		do(xgoesi.NewContextWithForceRefresh(context.Background()))
		assert.Equal(t, 2, calls, "expected force refresh to bypass the cache")
	})
}
