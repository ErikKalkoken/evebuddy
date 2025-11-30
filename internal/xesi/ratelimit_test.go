package xesi_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/xesi"
	"github.com/antihax/goesi"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	t.Run("should pass through request for operation with no rate limit", func(t *testing.T) {
		t.Parallel()
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
		})
		ts := httptest.NewServer(mockHandler)
		defer ts.Close()
		client := &http.Client{
			Transport: xesi.NewRateLimiter(),
		}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
	t.Run("should fail when operation has rate limit and is authed and characterID is missing", func(t *testing.T) {
		t.Parallel()
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
		})
		ts := httptest.NewServer(mockHandler)
		defer ts.Close()
		client := &http.Client{
			Transport: xesi.NewRateLimiter(),
		}
		ctx := context.WithValue(context.Background(), goesi.ContextAccessToken, "token")
		ctx = xesi.NewContextWithOperationID(ctx, "op")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		_, gotErr := client.Do(req)
		assert.Error(t, gotErr)
	})
	t.Run("should fail when operationID is unknown", func(t *testing.T) {
		t.Parallel()
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
		})
		ts := httptest.NewServer(mockHandler)
		defer ts.Close()
		client := &http.Client{
			Transport: xesi.NewRateLimiter(),
		}
		ctx := xesi.NewContextWithAuth(context.Background(), 42, "token")
		ctx = xesi.NewContextWithOperationID(ctx, "op")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		_, gotErr := client.Do(req)
		assert.Error(t, gotErr)
	})
	t.Run("should limit subsequent requests to the same bucket", func(t *testing.T) {
		t.Parallel()
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
		})
		ts := httptest.NewServer(mockHandler)
		defer ts.Close()
		client := &http.Client{
			Transport: xesi.NewRateLimiter(),
		}
		ctx := xesi.NewContextWithAuth(context.Background(), 42, "token")
		ctx = xesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdLocation")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		start := time.Now()
		for range 2 {
			resp, err := client.Do(req)
			if !assert.NoError(t, err) {
				t.Fatal(err)
			}
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}
		end := time.Since(start)
		assert.Greater(t, end, 1500*time.Millisecond)
	})
	t.Run("should limit concurrent requests", func(t *testing.T) {
		t.Parallel()
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
		})
		ts := httptest.NewServer(mockHandler)
		defer ts.Close()
		client := &http.Client{
			Transport: xesi.NewRateLimiter(),
		}
		ctx := xesi.NewContextWithAuth(context.Background(), 42, "token")
		ctx = xesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdLocation")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() {
				resp, err := client.Do(req)
				if !assert.NoError(t, err) {
					t.Fatal(err)
				}
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			})
		}
		start := time.Now()
		wg.Wait()
		end := time.Since(start)
		assert.Greater(t, end, 1500*time.Millisecond)
	})
	t.Run("should not limit concurrent requests from different characters", func(t *testing.T) {
		t.Parallel()
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
		})
		ts := httptest.NewServer(mockHandler)
		defer ts.Close()
		client := &http.Client{
			Transport: xesi.NewRateLimiter(),
		}
		var wg sync.WaitGroup
		for _, characterID := range []int32{42, 43} {
			wg.Go(func() {
				ctx := xesi.NewContextWithAuth(context.Background(), characterID, "token")
				ctx = xesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdLocation")
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
				if !assert.NoError(t, err) {
					t.Fatal(err)
				}
				resp, err := client.Do(req)
				if !assert.NoError(t, err) {
					t.Fatal(err)
				}
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			})
		}
		start := time.Now()
		wg.Wait()
		end := time.Since(start)
		assert.Less(t, end, 500*time.Millisecond)
	})
	t.Run("should not limit concurrent requests from same character for different rate groups", func(t *testing.T) {
		t.Parallel()
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
		})
		ts := httptest.NewServer(mockHandler)
		defer ts.Close()
		client := &http.Client{
			Transport: xesi.NewRateLimiter(),
		}
		var wg sync.WaitGroup
		for _, operationID := range []string{"GetCharactersCharacterIdLocation", "GetCharactersCharacterIdNotifications"} {
			wg.Go(func() {
				ctx := xesi.NewContextWithAuth(context.Background(), 42, "token")
				ctx = xesi.NewContextWithOperationID(ctx, operationID)
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
				if !assert.NoError(t, err) {
					t.Fatal(err)
				}
				resp, err := client.Do(req)
				if !assert.NoError(t, err) {
					t.Fatal(err)
				}
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			})
		}
		start := time.Now()
		wg.Wait()
		end := time.Since(start)
		assert.Less(t, end, 500*time.Millisecond)
	})
}
