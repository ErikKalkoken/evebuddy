package settings

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestCalcEarliest(t *testing.T) {
	f := func(v time.Time) string {
		return v.Format(time.RFC3339)
	}
	// earliest and want are computed from "now" at subtest execution time (not once
	// up front) since each subtest sets up a fresh in-memory DB with its own
	// migrations, whose duration can vary enough under load to drift a
	// pre-computed timestamp past the assertion's tolerance below.
	cases := []struct {
		name         string
		earliest     func(now time.Time) string
		timeoutHours int
		shouldSet    bool
		want         func(now time.Time) time.Time
	}{
		{"earliest after timeout", func(now time.Time) string { return f(now.Add(-1 * time.Hour)) }, 15 * 24, false, func(now time.Time) time.Time { return now.Add(-1 * time.Hour) }},
		{"earliest before timeout", func(now time.Time) string { return f(now.Add(-60 * 24 * time.Hour)) }, 15 * 24, false, func(now time.Time) time.Time { return now.Add(-15 * 24 * time.Hour) }},
		{"earliest before timeout fallback", func(now time.Time) string { return f(now.Add(-60 * 24 * time.Hour)) }, 0, false, func(now time.Time) time.Time { return now.Add(-settingNotifyTimeoutHoursDefault * time.Hour) }},
		{"timeout not set", func(now time.Time) string { return f(now.Add(-60 * 24 * time.Hour)) }, 0, false, func(now time.Time) time.Time { return now.Add(-settingNotifyTimeoutHoursDefault * time.Hour) }},
		{"earliest not set", func(now time.Time) string { return "" }, 15 * 2, true, func(now time.Time) time.Time { return now.Add(-notifyEarliestFallback) }},
		{"nothing set", func(now time.Time) string { return "" }, 0, true, func(now time.Time) time.Time { return now.Add(-notifyEarliestFallback) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			_, st, _ := testutil.NewDBInMemory()
			s, err := New(context.Background(), st)
			require.NoError(t, err)
			now := time.Now().UTC()
			if earliest := tc.earliest(now); earliest != "" {
				s.values["earliest"] = earliest
			}
			if tc.timeoutHours != 0 {
				s.values[settingNotifyTimeoutHours] = strconv.Itoa(tc.timeoutHours)
			}
			// when
			v := s.calcNotifyEarliest("earliest")
			// then
			want := tc.want(now)
			assert.WithinDuration(t, want, v, 5*time.Second)
			if tc.shouldSet {
				assert.Equal(t, want.Format(time.RFC3339), s.values["earliest"])
			}
		})
	}
}

func TestRecentSearches(t *testing.T) {
	t.Run("skips malformed entries", func(t *testing.T) {
		s := newTestSettings(t)
		s.values[settingRecentSearches] = `["123","not-a-number","456"]`
		got := s.RecentSearches()
		assert.Equal(t, []int64{123, 456}, got)
	})
}
