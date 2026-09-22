package xgoesi_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohugoio/httpcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func TestShouldCacheGetOrHeadOnly(t *testing.T) {
	newRequest := func(t *testing.T, method string) *http.Request {
		req, err := http.NewRequest(method, "https://esi.evetech.net/characters/affiliation", nil)
		require.NoError(t, err)
		return req
	}
	cases := []struct {
		method string
		want   bool
	}{
		{http.MethodGet, true},
		{http.MethodHead, true},
		{http.MethodPost, false},
		{http.MethodPut, false},
		{http.MethodPatch, false},
		{http.MethodDelete, false},
	}
	for _, c := range cases {
		t.Run(c.method, func(t *testing.T) {
			req := newRequest(t, c.method)
			got := xgoesi.ShouldCacheGetOrHeadOnly(req, &http.Response{}, "key")
			assert.Equal(t, c.want, got)
		})
	}
}

func TestShouldCacheGetOrHeadOnly_IntegratesWithHttpcache(t *testing.T) {
	// Regression test: two POSTs with different bodies to the same URL must not share a cached response.
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
		Cache:       newMemoryCache(),
		CacheKey:    xgoesi.CacheKeyWithForceRefresh,
		ShouldCache: xgoesi.ShouldCacheGetOrHeadOnly,
	}
	client := &http.Client{Transport: transport}
	do := func(method string) {
		req, err := http.NewRequestWithContext(context.Background(), method, ts.URL, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		_, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
	}

	do(http.MethodPost)
	assert.Equal(t, 1, calls)

	// a second POST must not be served from a cached response
	do(http.MethodPost)
	assert.Equal(t, 2, calls, "expected POST requests to never be served from cache")

	// GET is still cached as normal
	do(http.MethodGet)
	assert.Equal(t, 3, calls)
	do(http.MethodGet)
	assert.Equal(t, 3, calls, "expected GET request to be served from cache")
}
