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
	now := time.Now().UTC()
	f := func(v time.Time) string {
		return v.Format(time.RFC3339)
	}
	earliestFallback := now.Add(-notifyEarliestFallback)
	timeoutDefault := now.Add(-settingNotifyTimeoutHoursDefault * time.Hour)
	cases := []struct {
		name         string
		earliest     string
		timeoutHours int
		shouldSet    bool
		want         time.Time
	}{
		{"earliest after timeout", f(now.Add(-1 * time.Hour)), 15 * 24, false, now.Add(-1 * time.Hour)},
		{"earliest before timeout", f(now.Add(-60 * 24 * time.Hour)), 15 * 24, false, now.Add(-15 * 24 * time.Hour)},
		{"earliest before timeout fallback", f(now.Add(-60 * 24 * time.Hour)), 0, false, timeoutDefault},
		{"timeout not set", f(now.Add(-60 * 24 * time.Hour)), 0, false, timeoutDefault},
		{"earliest not set", "", 15 * 2, true, earliestFallback},
		{"nothing set", "", 0, true, earliestFallback},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			_, st, _ := testutil.NewDBInMemory()
			s, err := New(context.Background(), st)
			require.NoError(t, err)
			if tc.earliest != "" {
				s.values["earliest"] = tc.earliest
			}
			if tc.timeoutHours != 0 {
				s.values[settingNotifyTimeoutHours] = strconv.Itoa(tc.timeoutHours)
			}
			// when
			v := s.calcNotifyEarliest("earliest")
			// then
			assert.WithinDuration(t, tc.want, v, 5*time.Second)
			if tc.shouldSet {
				assert.Equal(t, earliestFallback.Format(time.RFC3339), s.values["earliest"])
			}
		})
	}
}
