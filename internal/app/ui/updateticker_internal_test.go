package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"github.com/stretchr/testify/assert"
)

type myPref struct {
	fyne.Preferences

	earliest        string
	timeout         int
	timeoutFallback int
}

func (p myPref) IntWithFallback(key string, fallback int) int {
	if p.timeoutFallback != 0 {
		return p.timeoutFallback
	}
	return p.timeout
}

func (p myPref) String(k string) string {
	return p.earliest
}

func TestCalcEarliest(t *testing.T) {
	now := time.Now().UTC()
	f := func(v time.Time) string {
		return v.Format(time.RFC3339)
	}
	cases := []struct {
		name                 string
		earliest             string
		timeoutHours         int
		timeoutHoursFallback int
		want                 time.Time
	}{
		{"earliest after timeout", f(now.Add(-1 * time.Hour)), 30 * 24, 0, now.Add(-1 * time.Hour)},
		{"earliest before timeout", f(now.Add(-60 * 24 * time.Hour)), 30 * 24, 0, now.Add(-30 * 24 * time.Hour)},
		{"earliest before timeout fallback", f(now.Add(-60 * 24 * time.Hour)), 0, 30 * 24, now.Add(-30 * 24 * time.Hour)},
		{"timeout not set", f(now.Add(-60 * 24 * time.Hour)), 0, 0, now.Add(-60 * 24 * time.Hour)},
		{"earliest not set", "", 30 * 24, 0, now.Add(-30 * 24 * time.Hour)},
		{"nothing set", "", 0, 0, time.Time{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := myPref{
				earliest:        tc.earliest,
				timeout:         tc.timeoutHours,
				timeoutFallback: tc.timeoutHoursFallback,
			}
			x := calcNotifyEarliest(p, "alpha")
			assert.WithinDuration(t, tc.want, x, 5*time.Second)
		})
	}
}
