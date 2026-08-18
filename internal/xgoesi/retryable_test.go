package xgoesi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestShouldRetryOn420s(t *testing.T) {
	responses := []struct {
		statusCode int
		remain     int
		reset      int
		body       string
	}{
		{
			http.StatusOK,
			100,
			60,
			"dummy",
		},
		{
			xgoesi.StatusTooManyErrors,
			0,
			1,
			"dummy",
		},
	}
	var callCount int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, ok := xslices.Pop(&responses)
		if !ok {
			t.Fatal("out of test reponses")
		}
		w.Header().Set("X-ESI-Error-Limit-Remain", strconv.Itoa(resp.remain))
		w.Header().Set("X-ESI-Error-Limit-Reset", strconv.Itoa(resp.reset))
		w.WriteHeader(resp.statusCode)
		fmt.Fprint(w, resp.body)
		callCount++
	}))
	defer ts.Close()
	client := retryablehttp.NewClient()
	client.RetryMax = 1
	client.CheckRetry = xgoesi.CustomCheckRetry
	client.Backoff = xgoesi.CustomBackoff
	client.HTTPClient.Transport = &xgoesi.RateLimiter{}
	resp, err := client.Get(ts.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, callCount)
}

func TestShouldRetryOn429s(t *testing.T) {
	responses := []struct {
		statusCode int
		retryAfter int
		body       string
	}{
		{
			http.StatusOK,
			0,
			"dummy",
		},
		{
			http.StatusTooManyRequests,
			1,
			"dummy",
		},
	}
	var callCount int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, ok := xslices.Pop(&responses)
		if !ok {
			t.Fatal("out of test reponses")
		}
		if resp.retryAfter > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(resp.retryAfter))
		}
		w.WriteHeader(resp.statusCode)
		fmt.Fprint(w, resp.body)
		callCount++
	}))
	defer ts.Close()
	client := retryablehttp.NewClient()
	client.RetryMax = 1
	client.CheckRetry = xgoesi.CustomCheckRetry
	client.Backoff = xgoesi.CustomBackoff
	client.HTTPClient.Transport = &xgoesi.RateLimiter{}
	resp, err := client.Get(ts.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, callCount)
}

func TestShouldNotRetryOn520s(t *testing.T) {
	var callCount int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(xgoesi.StatusUnknownError)
		w.Write([]byte("Hello, World!"))
		callCount++
	}))
	defer ts.Close()
	client := retryablehttp.NewClient()
	client.RetryMax = 1
	client.CheckRetry = xgoesi.CustomCheckRetry
	client.Backoff = xgoesi.CustomBackoff
	resp, err := client.Get(ts.URL)
	require.NoError(t, err)
	assert.Equal(t, xgoesi.StatusUnknownError, resp.StatusCode)
	assert.Equal(t, 1, callCount)
}
