package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/pcache"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObfuscate(t *testing.T) {
	cases := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{"normal", "123456789", 4, "XXXXX6789"},
		{"s too short", "123", 4, "XXX"},
		{"n is zero", "123456789", 0, "XXXXXXXXX"},
		{"n is negative", "123456789", -5, "XXXXXXXXX"},
		{"s is empty", "", 4, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := obfuscate(tc.s, tc.n, "X")
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCacheAdapter(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	pc := pcache.New(st, 0)
	ca := newCacheAdapter(pc, "prefix", 0)
	t.Run("get existing key", func(t *testing.T) {
		pc.Clear()
		ca.Set("a", []byte("alpha"))
		got, ok := ca.Get("a")
		if assert.True(t, ok) {
			assert.Equal(t, []byte("alpha"), got)
		}
	})
	t.Run("get non existing key", func(t *testing.T) {
		pc.Clear()
		_, ok := ca.Get("a")
		assert.False(t, ok)
	})
	t.Run("delete existing key", func(t *testing.T) {
		pc.Clear()
		ca.Set("a", []byte("alpha"))
		ca.Delete("a")
		_, ok := ca.Get("a")
		assert.False(t, ok)
	})
}

func TestSetupCrashFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), crashFileName)
	err := setupCrashFile(p)
	if assert.NoError(t, err) {
		_, err := os.Stat(p)
		assert.NoError(t, err)
	}
}

func TestRetryOn420s(t *testing.T) {
	responses := []struct {
		statusCode int
		reset      string
		body       string
	}{
		{
			http.StatusOK,
			"60",
			"dummy",
		},
		{
			xgoesi.StatusTooManyErrors,
			"1",
			"dummy",
		},
	}
	var callCount int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, ok := xslices.Pop(&responses)
		if !ok {
			t.Fatal("out of test reponses")
		}
		w.Header().Set("X-ESI-Error-Limit-Reset", resp.reset)
		w.WriteHeader(resp.statusCode)
		fmt.Fprint(w, resp.body)
		callCount++
	}))
	defer ts.Close()
	client := retryablehttp.NewClient()
	client.RetryMax = 1
	client.CheckRetry = customCheckRetry
	client.Backoff = customBackoff
	client.HTTPClient.Transport = xgoesi.NewRateLimiter()
	resp, err := client.Get(ts.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, callCount)
}
