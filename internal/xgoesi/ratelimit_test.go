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
			Transport: &xgoesi.RateLimiter{},
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
			Transport: &xgoesi.RateLimiter{},
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
			Transport: &xgoesi.RateLimiter{},
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
			Transport: &xgoesi.RateLimiter{},
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
			Transport: &xgoesi.RateLimiter{},
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
			Transport: &xgoesi.RateLimiter{},
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
	const (
		headerErrorLimitRemain = "X-ESI-Error-Limit-Remain"
		headerErrorLimitReset  = "X-ESI-Error-Limit-Reset"
	)
	t.Run("should pass through request when error limit is not exceeded", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(headerErrorLimitRemain, "100")
			w.Header().Set(headerErrorLimitReset, "55")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
		}))
		defer ts.Close()
		client := &http.Client{
			Transport: &xgoesi.RateLimiter{},
		}
		resp, err := client.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
	t.Run("should passs through 420 response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(headerErrorLimitRemain, "0")
			w.Header().Set(headerErrorLimitReset, "55")
			w.WriteHeader(xgoesi.StatusTooManyErrors)
			fmt.Fprint(w, "dummy")
		}))
		defer ts.Close()
		client := &http.Client{
			Transport: &xgoesi.RateLimiter{},
		}
		resp, err := client.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, xgoesi.StatusTooManyErrors, resp.StatusCode)
	})
	t.Run("should respond with synthetic 420 when timeout still active", func(t *testing.T) {
		responses := []struct {
			statusCode int
			remain     int
			reset      int
			body       string
		}{
			{
				http.StatusOK,
				100,
				55,
				"dummy",
			},
			{
				xgoesi.StatusTooManyErrors,
				0,
				55,
				"dummy",
			},
		}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			resp, ok := xslices.Pop(&responses)
			if !ok {
				t.Fatal("out of test reponses")
			}
			w.Header().Set(headerErrorLimitRemain, strconv.Itoa(resp.remain))
			w.Header().Set(headerErrorLimitReset, strconv.Itoa(resp.reset))
			w.WriteHeader(resp.statusCode)
			fmt.Fprint(w, resp.body)
		}))
		defer ts.Close()
		// first request will set the 420 ban
		client := &http.Client{
			Transport: &xgoesi.RateLimiter{},
		}
		resp, err := client.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, xgoesi.StatusTooManyErrors, resp.StatusCode)
		// second request will return synthetic 420 during active timeout
		resp, err = client.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, xgoesi.StatusTooManyErrors, resp.StatusCode)
		reset, err := strconv.Atoi(resp.Header.Get(headerErrorLimitReset))
		require.NoError(t, err)
		assert.Equal(t, reset, 55)
		remain, err := strconv.Atoi(resp.Header.Get(headerErrorLimitRemain))
		require.NoError(t, err)
		assert.Equal(t, remain, 0)
	})
	t.Run("should block subsequent request in the current window when error threshold reached", func(t *testing.T) {
		responses := []struct {
			statusCode int
			remain     int
			reset      int
			body       string
		}{
			{
				http.StatusOK,
				100,
				55,
				"dummy",
			},
			{
				http.StatusOK,
				5,
				55,
				"dummy",
			},
		}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			resp, ok := xslices.Pop(&responses)
			if !ok {
				t.Fatal("out of test reponses")
			}
			w.Header().Set(headerErrorLimitRemain, strconv.Itoa(resp.remain))
			w.Header().Set(headerErrorLimitReset, strconv.Itoa(resp.reset))
			w.WriteHeader(resp.statusCode)
			fmt.Fprint(w, resp.body)
		}))
		defer ts.Close()
		// first request will set the 420 timeout
		client := &http.Client{
			Transport: &xgoesi.RateLimiter{},
		}
		resp, err := client.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		// second request will return synthetic 420 during active timeout
		resp, err = client.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, xgoesi.StatusTooManyErrors, resp.StatusCode)
		reset, err := strconv.Atoi(resp.Header.Get(headerErrorLimitReset))
		require.NoError(t, err)
		assert.Equal(t, reset, 55)
		remain, err := strconv.Atoi(resp.Header.Get(headerErrorLimitRemain))
		require.NoError(t, err)
		assert.Equal(t, remain, 0)
	})
}
