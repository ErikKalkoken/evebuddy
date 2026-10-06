package screens

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
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
