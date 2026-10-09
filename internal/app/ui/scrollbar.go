package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

const wideScrollBarSize = 18

// wideScrollBarTheme widens the thin scroll bar so it can be dragged on mobile.
// It must not use theme.Current(), which returns this override while rendering.
type wideScrollBarTheme struct{}

func (wideScrollBarTheme) base() fyne.Theme {
	return fyne.CurrentApp().Settings().Theme()
}

func (t wideScrollBarTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return t.base().Color(n, v)
}

func (t wideScrollBarTheme) Font(s fyne.TextStyle) fyne.Resource {
	return t.base().Font(s)
}

func (t wideScrollBarTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return t.base().Icon(n)
}

func (t wideScrollBarTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameScrollBar, theme.SizeNameScrollBarSmall:
		return wideScrollBarSize
	}
	return t.base().Size(n)
}

// NewWideScrollBarOverride returns content with wide scroll bars, which are easier to drag on mobile.
func NewWideScrollBarOverride(content fyne.CanvasObject) fyne.CanvasObject {
	return container.NewThemeOverride(content, wideScrollBarTheme{})
}
