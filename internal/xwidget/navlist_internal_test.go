package xwidget

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
)

func TestNavListItem_Style(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	color := func(it *NavListItem, name fyne.ThemeColorName) any {
		return it.Theme().Color(name, fyne.CurrentApp().Settings().ThemeVariant())
	}
	makeItem := func(t *testing.T) *NavListItem {
		it := NewNavListItem("Headline", theme.HomeIcon(), nil)
		it.Supporting = "Supporting"
		w := test.NewWindow(it)
		t.Cleanup(w.Close)
		return it
	}

	t.Run("should show enabled item with muted supporting text", func(t *testing.T) {
		it := makeItem(t)
		it.Refresh()
		assert.Equal(t, color(it, theme.ColorNameForeground), it.headlineText.Color)
		assert.Equal(t, color(it, theme.ColorNamePlaceHolder), it.supportingText.Color)
		assert.IsType(t, &theme.ThemedResource{}, it.leadingImage.Resource)
	})
	t.Run("should show disabled item in disabled color", func(t *testing.T) {
		it := makeItem(t)
		it.IsDisabled = true
		it.Refresh()
		assert.Equal(t, color(it, theme.ColorNameDisabled), it.headlineText.Color)
		assert.Equal(t, color(it, theme.ColorNameDisabled), it.supportingText.Color)
		assert.IsType(t, &theme.DisabledResource{}, it.leadingImage.Resource)
	})
	t.Run("should restore style when enabled again", func(t *testing.T) {
		it := makeItem(t)
		it.IsDisabled = true
		it.Refresh()
		it.IsDisabled = false
		it.Refresh()
		assert.Equal(t, color(it, theme.ColorNameForeground), it.headlineText.Color)
		assert.Equal(t, color(it, theme.ColorNamePlaceHolder), it.supportingText.Color)
		assert.IsType(t, &theme.ThemedResource{}, it.leadingImage.Resource)
	})
	t.Run("should apply style when refreshed before first render", func(t *testing.T) {
		it := NewNavListItem("Headline", theme.HomeIcon(), nil)
		it.IsDisabled = true
		it.Refresh()
		w := test.NewWindow(it)
		defer w.Close()
		assert.Equal(t, color(it, theme.ColorNameDisabled), it.headlineText.Color)
		assert.IsType(t, &theme.DisabledResource{}, it.leadingImage.Resource)
	})
}

func TestNavListItem_Visibility(t *testing.T) {
	test.NewTempApp(t)
	makeItem := func(t *testing.T, leading fyne.Resource) *NavListItem {
		it := NewNavListItem("Headline", leading, nil)
		w := test.NewWindow(it)
		t.Cleanup(w.Close)
		return it
	}

	t.Run("should hide empty slots", func(t *testing.T) {
		it := makeItem(t, nil)
		assert.False(t, it.leadingWrapped.Visible())
		assert.False(t, it.trailingWrapped.Visible())
		assert.False(t, it.supportingText.Visible())
	})
	t.Run("should show slots when set", func(t *testing.T) {
		it := makeItem(t, theme.HomeIcon())
		it.Trailing = theme.InfoIcon()
		it.Supporting = "Supporting"
		it.Refresh()
		assert.True(t, it.leadingWrapped.Visible())
		assert.True(t, it.trailingWrapped.Visible())
		assert.True(t, it.supportingText.Visible())
	})
	t.Run("should hide slots again when cleared", func(t *testing.T) {
		it := makeItem(t, theme.HomeIcon())
		it.Trailing = theme.InfoIcon()
		it.Supporting = "Supporting"
		it.Refresh()
		it.Trailing = nil
		it.Supporting = ""
		it.Refresh()
		assert.False(t, it.trailingWrapped.Visible())
		assert.False(t, it.supportingText.Visible())
	})
}
