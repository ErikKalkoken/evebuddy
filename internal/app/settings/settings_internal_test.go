package settings

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"github.com/stretchr/testify/assert"
)

// myPreferences represents a stub for replacing fyne.Preferences in tests.
type myPreferences struct {
	fyne.Preferences

	data map[string]any
}

func NewMyPref() myPreferences {
	p := myPreferences{data: map[string]any{}}
	return p
}

func (p myPreferences) Bool(key string) bool {
	return getAny[bool](p, key)
}

func (p myPreferences) BoolWithFallback(key string, fallback bool) bool {
	return getAnyWithFallback(p, key, fallback)
}

func (p myPreferences) SetBool(k string, v bool) {
	setAny(p, k, v)
}

func (p myPreferences) Int(key string) int {
	return getAny[int](p, key)
}

func (p myPreferences) IntWithFallback(key string, fallback int) int {
	return getAnyWithFallback(p, key, fallback)
}

func (p myPreferences) SetInt(k string, v int) {
	setAny(p, k, v)
}

func (p myPreferences) String(key string) string {
	return getAny[string](p, key)
}

func (p myPreferences) StringWithFallback(key string, fallback string) string {
	return getAnyWithFallback(p, key, fallback)
}

func (p myPreferences) SetString(k string, v string) {
	setAny(p, k, v)
}

func (p myPreferences) StringList(key string) []string {
	return getAny[[]string](p, key)
}

func (p myPreferences) StringListWithFallback(key string, fallback []string) []string {
	return getAnyWithFallback(p, key, fallback)
}

func (p myPreferences) SetStringList(k string, v []string) {
	setAny(p, k, v)
}

func (p myPreferences) FloatList(key string) []float64 {
	return getAny[[]float64](p, key)
}

func (p myPreferences) SetFloatList(k string, v []float64) {
	setAny(p, k, v)
}

func (p myPreferences) IntList(key string) []int {
	return getAny[[]int](p, key)
}

func (p myPreferences) IntListWithFallback(key string, fallback []int) []int {
	return getAnyWithFallback(p, key, fallback)
}

func (p myPreferences) SetIntList(k string, v []int) {
	setAny(p, k, v)
}

func getAny[T any](p myPreferences, k string) T {
	var z T
	return getAnyWithFallback(p, k, z)
}

func getAnyWithFallback[T any](p myPreferences, key string, fallback T) T {
	x, ok := p.data[key]
	if !ok {
		return fallback
	}
	v, ok := x.(T)
	if !ok {
		return fallback
	}
	return v
}
func setAny(p myPreferences, k string, v any) {
	p.data[k] = v
}

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
			p := NewMyPref()
			p.data["earliest"] = tc.earliest
			if tc.timeoutHours != 0 {
				p.data[settingNotifyTimeoutHours] = tc.timeoutHours
			}
			s := New(p)
			// when
			v := s.calcNotifyEarliest("earliest")
			// then
			assert.WithinDuration(t, tc.want, v, 5*time.Second)
			if tc.shouldSet {
				assert.Equal(t, earliestFallback.Format(time.RFC3339), p.data["earliest"])
			}
		})
	}
}
