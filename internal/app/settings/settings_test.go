package settings_test

import (
	"log/slog"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestSettings(t *testing.T) {
	t.Run("Window size", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		x := fyne.NewSize(123, 456)
		s.SetWindowSize(x)
	xassert.Equal(t, x, s.WindowSize())
	})
	t.Run("Log level", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		x := "debug"
		s.SetLogLevel(x)
	xassert.Equal(t, x, s.LogLevel())
	xassert.Equal(t, slog.LevelDebug, s.LogLevelSlog())
	})
	t.Run("RecentSearches", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		x := []int64{1, 2, 3}
		s.SetRecentSearches(x)
	xassert.Equal(t, x, s.RecentSearches())
	})
	t.Run("NotificationTypesEnabled", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		got := set.Of([]string{"alpha", "bravo"}...)
		s.SetNotificationTypesEnabled(got)
		want := s.NotificationTypesEnabled()
		xassert.Equal(t, want, got)
	})
}

func TestColorTheme(t *testing.T) {
	t.Run("Default theme", func(t *testing.T) {
		s := settings.New(settings.NewMyPref())
		x1 := s.ColorTheme()
	xassert.Equal(t, settings.Auto, x1)
	})
	t.Run("Can set and get theme", func(t *testing.T) {
		s := settings.New(settings.NewMyPref())
		s.SetColorTheme(settings.Dark)
		x1 := s.ColorTheme()
	xassert.Equal(t, settings.Dark, x1)
	})
}
