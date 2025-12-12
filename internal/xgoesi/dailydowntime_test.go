package xgoesi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDailyDowntime(t *testing.T) {
	original := xgoesi.TimeNow
	defer func() {
		xgoesi.TimeNow = original
	}()

	cases := []struct {
		now  time.Time
		want bool
	}{
		{time.Date(2025, 12, 1, 11, 1, 0, 0, time.UTC), true},
		{time.Date(2025, 12, 1, 10, 1, 0, 0, time.UTC), false},
	}
	for _, tc := range cases {
		xgoesi.TimeNow = func() time.Time {
			return tc.now
		}
		assert.Equal(t, tc.want, xgoesi.IsDailyDowntime())
	}
}

func TestDailyDowntime(t *testing.T) {
	original := xgoesi.TimeNow
	defer func() {
		xgoesi.TimeNow = original
	}()
	xgoesi.TimeNow = func() time.Time {
		return time.Date(2025, 12, 1, 11, 1, 0, 0, time.UTC)
	}
	start, finish := xgoesi.DailyDowntime()
	assert.Equal(t, time.Date(2025, 12, 1, 11, 0, 0, 0, time.UTC), start)
	assert.Equal(t, time.Date(2025, 12, 1, 11, 15, 0, 0, time.UTC), finish)
}

func TestDowntimeBlocker(t *testing.T) {
	t.Run("should pass through outside of daily downtime", func(t *testing.T) {
		original := xgoesi.TimeNow
		defer func() {
			xgoesi.TimeNow = original
		}()
		xgoesi.TimeNow = func() time.Time {
			return time.Date(2025, 12, 1, 10, 10, 0, 0, time.UTC)
		}
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
		})
		ts := httptest.NewServer(mockHandler)
		defer ts.Close()
		client := &http.Client{
			Transport: &xgoesi.DowntimeBlocker{},
		}
		resp, err := client.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
	t.Run("should block during daily downtime", func(t *testing.T) {
		original := xgoesi.TimeNow
		defer func() {
			xgoesi.TimeNow = original
		}()
		xgoesi.TimeNow = func() time.Time {
			return time.Date(2025, 12, 1, 11, 10, 0, 0, time.UTC)
		}
		mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Hello, Mock Server!")
		})
		ts := httptest.NewServer(mockHandler)
		defer ts.Close()
		client := &http.Client{
			Transport: &xgoesi.DowntimeBlocker{},
		}
		resp, err := client.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		assert.Equal(t, "301", resp.Header.Get("Retry-After"))
	})
}
