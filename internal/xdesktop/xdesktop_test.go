package xdesktop_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
)

// TODO: Complete test suite

func TestShortcuts(t *testing.T) {
	cs := &desktop.CustomShortcut{
		KeyName:  fyne.KeyC,
		Modifier: fyne.KeyModifierAlt,
	}
	app := test.NewTempApp(t)
	w := app.NewWindow("Dummy")
	sc1 := xdesktop.ShortcutWithHandler{
		Shortcut: cs,
		Handler: func(shortcut fyne.Shortcut) {
			panic("TODO")
		},
	}
	t.Run("can add a shortcut", func(t *testing.T) {
		// given
		xdesktop.RemoveAllShortcuts(w)
		// when
		xdesktop.AddShortcut("alpha", sc1, w)
		// then
		sc2, ok := xdesktop.Shortcut("alpha", w)
		require.True(t, ok)
		xassert.Equal(t, sc1.Shortcut, sc2.Shortcut)
	})
	t.Run("can remove a shortcut", func(t *testing.T) {
		// given
		xdesktop.RemoveAllShortcuts(w)
		xdesktop.AddShortcut("alpha", sc1, w)
		// when
		sc2 := xdesktop.RemoveShortcut("alpha", w)
		// then
		xassert.Equal(t, sc1.Shortcut, sc2.Shortcut)
		_, ok := xdesktop.Shortcut("alpha", w)
		assert.False(t, ok)
	})
}
