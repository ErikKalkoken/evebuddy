package settings_test

import (
	"log/slog"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/stretchr/testify/assert"
)

func TestSettings(t *testing.T) {
	t.Run("Window size", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		x := fyne.NewSize(123, 456)
		s.SetWindowSize(x)
		assert.Equal(t, x, s.WindowSize())
	})
	t.Run("Log level", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		x := "debug"
		s.SetLogLevel(x)
		assert.Equal(t, x, s.LogLevel())
		assert.Equal(t, slog.LevelDebug, s.LogLevelSlog())
	})
	t.Run("RecentSearches", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		x := []int32{1, 2, 3}
		s.SetRecentSearches(x)
		assert.Equal(t, x, s.RecentSearches())
	})
	t.Run("NotificationTypesEnabled", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		got := set.Of([]string{"alpha", "bravo"}...)
		s.SetNotificationTypesEnabled(got)
		want := s.NotificationTypesEnabled()
		xassert.EqualSet(t, want, got)
	})
}

func TestColorTheme(t *testing.T) {
	t.Run("Default theme", func(t *testing.T) {
		s := settings.New(settings.NewMyPref())
		x1 := s.ColorTheme()
		assert.Equal(t, settings.Auto, x1)
	})
	t.Run("Can set and get theme", func(t *testing.T) {
		s := settings.New(settings.NewMyPref())
		s.SetColorTheme(settings.Dark)
		x1 := s.ColorTheme()
		assert.Equal(t, settings.Dark, x1)
	})
}
