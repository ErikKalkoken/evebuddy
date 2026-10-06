package screens

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColonyPinIcon(t *testing.T) {
	test.NewTempApp(t)
	tinted := func(w *colonyPinIcon) fyne.Resource {
		c := w.Theme().Color(theme.ColorNamePlaceHolder, fyne.CurrentApp().Settings().ThemeVariant())
		return colonyTintedIcon(w.icon, color.NRGBAModel.Convert(c).(color.NRGBA))
	}

	t.Run("should show tinted icon when created", func(t *testing.T) {
		w := newColonyPinIcon(pinTypeExtractor.icon(), theme.ColorNamePlaceHolder)
		assert.Equal(t, tinted(w), w.Resource)
	})
	t.Run("should tint icon again when theme changed", func(t *testing.T) {
		test.ApplyTheme(t, theme.DefaultTheme())
		w := newColonyPinIcon(pinTypeExtractor.icon(), theme.ColorNamePlaceHolder)
		win := test.NewWindow(w)
		t.Cleanup(win.Close)
		before := w.Resource
		test.ApplyTheme(t, test.Theme())
		w.Refresh()
		assert.Equal(t, tinted(w), w.Resource)
		assert.NotEqual(t, before, w.Resource)
	})
}

func TestColonyPlanetSymbol(t *testing.T) {
	test.NewTempApp(t)
	const iconID = 1047
	t.Run("should show planet in color without attention icon", func(t *testing.T) {
		w := newColonyPlanetSymbol(28, 1)
		w.set(iconID, false)
		assert.Equal(t, colonyPlanetIcon(iconID, false), w.icon.Resource)
		assert.False(t, w.attention.Visible())
	})
	t.Run("should show grayed out planet with attention icon for problems", func(t *testing.T) {
		w := newColonyPlanetSymbol(28, 1)
		w.set(iconID, true)
		require.NotEqual(t, colonyPlanetIcon(iconID, false), colonyPlanetIcon(iconID, true), "test icon has grayscale variant")
		assert.Equal(t, colonyPlanetIcon(iconID, true), w.icon.Resource)
		assert.True(t, w.attention.Visible())
	})
	t.Run("should restore planet when recycled for colony without problems", func(t *testing.T) {
		w := newColonyPlanetSymbol(28, 1)
		w.set(iconID, true)
		w.set(iconID, false)
		assert.Equal(t, colonyPlanetIcon(iconID, false), w.icon.Resource)
		assert.False(t, w.attention.Visible())
	})
	t.Run("should size attention icon in inline icon sizes", func(t *testing.T) {
		for _, scale := range []float32{1, 2} {
			w := newColonyPlanetSymbol(104, scale)
			win := test.NewWindow(w)
			w.set(iconID, true)
			assert.Equal(t, fyne.NewSquareSize(scale*theme.IconInlineSize()), w.attention.MinSize())
			win.Close()
		}
	})
}
