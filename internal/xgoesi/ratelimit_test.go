package xgoesi_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/antihax/goesi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestRateLimiter_RateLimited(t *testing.T) {
	t.Run("should fail when operation has rate limit and is authed and characterID is missing", func(t *testing.T) {
		t.Parallel()
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
		})
		ts := httptest.NewServer(mockHandler)
		defer ts.Close()
		client := &http.Client{
			Transport: xgoesi.NewRateLimiter(),
		}
		ctx := context.WithValue(t.Context(), goesi.ContextAccessToken, "token")
		ctx = xgoesi.NewContextWithOperationID(ctx, "op")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
		require.NoError(t, err)
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
			Transport: xgoesi.NewRateLimiter(),
		}
		ctx := xgoesi.NewContextWithAuth(t.Context(), 42, "token")
		ctx = xgoesi.NewContextWithOperationID(ctx, "op")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
		require.NoError(t, err)
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
			Transport: xgoesi.NewRateLimiter(),
		}
		ctx := xgoesi.NewContextWithAuth(t.Context(), 42, "token")
		ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdLocation")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
		require.NoError(t, err)
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
			Transport: xgoesi.NewRateLimiter(),
		}
		ctx := xgoesi.NewContextWithAuth(t.Context(), 42, "token")
		ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdLocation")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
		require.NoError(t, err)
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
			Transport: xgoesi.NewRateLimiter(),
		}
		var wg sync.WaitGroup
		for _, characterID := range []int32{42, 43} {
			wg.Go(func() {
				ctx := xgoesi.NewContextWithAuth(t.Context(), characterID, "token")
				ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdLocation")
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
			Transport: xgoesi.NewRateLimiter(),
		}
		var wg sync.WaitGroup
		for _, operationID := range []string{"GetCharactersCharacterIdLocation", "GetCharactersCharacterIdNotifications"} {
			wg.Go(func() {
				ctx := xgoesi.NewContextWithAuth(t.Context(), 42, "token")
				ctx = xgoesi.NewContextWithOperationID(ctx, operationID)
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

func TestRateLimiter_ErrorLimited(t *testing.T) {
	t.Run("should pass through request for operation with no rate limit", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
		}))
		defer ts.Close()
		client := &http.Client{
			Transport: xgoesi.NewRateLimiter(),
		}
		resp, err := client.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
	t.Run("should add retry-header to response when 420 is received", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-ESI-Error-Limit-Reset", "55")
			w.WriteHeader(xgoesi.StatusTooManyErrors)
			fmt.Fprint(w, "dummy")
		}))
		defer ts.Close()
		client := &http.Client{
			Transport: xgoesi.NewRateLimiter(),
		}
		resp, err := client.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, xgoesi.StatusTooManyErrors, resp.StatusCode)
		x2, err := strconv.Atoi(resp.Header.Get("Retry-After"))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, x2, 55)
	})
	t.Run("should reponse with synthetic 420 when timeout still ongoing", func(t *testing.T) {
		responses := []struct {
			statusCode int
			reset      string
			body       string
		}{
			{
				http.StatusOK,
				"",
				"dummy",
			},
			{
				xgoesi.StatusTooManyErrors,
				"55",
				"dummy",
			},
		}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			resp, ok := xslices.Pop(&responses)
			if !ok {
				t.Fatal("out of test reponses")
			}
			w.Header().Set("X-ESI-Error-Limit-Reset", resp.reset)
			w.WriteHeader(resp.statusCode)
			fmt.Fprint(w, resp.body)
		}))
		defer ts.Close()
		// first request will set the 420 ban
		client := &http.Client{
			Transport: xgoesi.NewRateLimiter(),
		}
		resp, err := client.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, xgoesi.StatusTooManyErrors, resp.StatusCode)
		x2, err := strconv.Atoi(resp.Header.Get("Retry-After"))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, x2, 55)
		// second request will return 420 from active block
		resp, err = client.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, xgoesi.StatusTooManyErrors, resp.StatusCode)
		x2, err = strconv.Atoi(resp.Header.Get("Retry-After"))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, x2, 55)
	})
}
