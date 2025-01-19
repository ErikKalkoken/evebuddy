package desktop

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/stretchr/testify/assert"
)

type myPref struct {
	fyne.Preferences

	data map[string]any
}

func (p myPref) IntWithFallback(key string, fallback int) int {
	x, ok := p.data[key]
	if !ok {
		return fallback
	}
	v, ok := x.(int)
	if !ok {
		return fallback
	}
	return v
}

func (p myPref) String(key string) string {
	x, ok := p.data[key]
	if !ok {
		return ""
	}
	v, ok := x.(string)
	if !ok {
		return ""
	}
	return v
}

func (p myPref) SetString(k, v string) {
	p.data[k] = v
}

func TestCalcEarliest(t *testing.T) {
	now := time.Now().UTC()
	f := func(v time.Time) string {
		return v.Format(time.RFC3339)
	}
	earliestFallback := now.Add(-notifyEarliestFallback)
	timeoutDefault := now.Add(-ui.SettingNotifyTimeoutHoursDefault * time.Hour)
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
			p := myPref{
				data: make(map[string]any),
			}
			p.data["earliest"] = tc.earliest
			if tc.timeoutHours != 0 {
				p.data[ui.SettingNotifyTimeoutHours] = tc.timeoutHours
			}
			// when
			v := calcNotifyEarliest(p, "earliest")
			// then
			assert.WithinDuration(t, tc.want, v, 5*time.Second)
			if tc.shouldSet {
				assert.Equal(t, earliestFallback.Format(time.RFC3339), p.data["earliest"])
			}
		})
	}
}
