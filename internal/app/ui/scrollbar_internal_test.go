package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestWideScrollBarTheme(t *testing.T) {
	test.NewTempApp(t)

	t.Run("widens scroll bars", func(t *testing.T) {
		// given
		th := wideScrollBarTheme{}
		// when/then
		xassert.Equal(t, float32(wideScrollBarSize), th.Size(theme.SizeNameScrollBar))
		xassert.Equal(t, float32(wideScrollBarSize), th.Size(theme.SizeNameScrollBarSmall))
	})
	t.Run("uses app theme for everything else", func(t *testing.T) {
		// given
		th := wideScrollBarTheme{}
		base := fyne.CurrentApp().Settings().Theme()
		// when/then
		xassert.Equal(t, base.Size(theme.SizeNamePadding), th.Size(theme.SizeNamePadding))
		xassert.Equal(t, base.Color(theme.ColorNamePrimary, theme.VariantDark), th.Color(theme.ColorNamePrimary, theme.VariantDark))
		xassert.Equal(t, base.Font(fyne.TextStyle{Bold: true}), th.Font(fyne.TextStyle{Bold: true}))
		xassert.Equal(t, base.Icon(theme.IconNameHome), th.Icon(theme.IconNameHome))
	})
}
