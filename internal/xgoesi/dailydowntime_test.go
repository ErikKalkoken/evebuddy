package xgoesi_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/stretchr/testify/assert"
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
