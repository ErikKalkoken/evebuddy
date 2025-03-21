package ui_test

import (
	"log/slog"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/stretchr/testify/assert"
)

func TestAppSettings(t *testing.T) {
	t.Run("Window size", func(t *testing.T) {
		p := ui.NewMyPref()
		s := ui.NewAppSettings(p)
		x := fyne.NewSize(123, 456)
		s.SetWindowSize(x)
		assert.Equal(t, x, s.WindowSize())
	})
	t.Run("Log level", func(t *testing.T) {
		p := ui.NewMyPref()
		s := ui.NewAppSettings(p)
		x := "debug"
		s.SetLogLevel(x)
		assert.Equal(t, x, s.LogLevel())
		assert.Equal(t, slog.LevelDebug, s.LogLevelSlog())
	})
	t.Run("NotificationTypesEnabled", func(t *testing.T) {
		p := ui.NewMyPref()
		s := ui.NewAppSettings(p)
		x := set.NewFromSlice([]string{"alpha", "bravo"})
		s.SetNotificationTypesEnabled(x)
		assert.Equal(t, x, s.NotificationTypesEnabled())
	})
}
