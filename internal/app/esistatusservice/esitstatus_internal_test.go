package esistatusservice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsDailyDowntime(t *testing.T) {
	start := "11h0m"
	end := "11h15m"
	cases := []struct {
		name string
		date time.Time
		want bool
	}{
		{"in range", time.Date(2025, 1, 1, 11, 3, 0, 0, time.UTC), true},
		{"equal start", time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC), true},
		{"equal end", time.Date(2025, 1, 1, 11, 15, 0, 0, time.UTC), true},
		{"after 1", time.Date(2025, 1, 1, 11, 16, 0, 0, time.UTC), false},
		{"after 2", time.Date(2025, 1, 1, 12, 10, 0, 0, time.UTC), false},
		{"before", time.Date(2025, 1, 1, 10, 16, 0, 0, time.UTC), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isDailyDowntime(start, end, tc.date)
			assert.Equal(t, tc.want, got)
		})
	}
}
