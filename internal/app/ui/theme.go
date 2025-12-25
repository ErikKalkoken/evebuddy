package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
)

const (
	colorNameInfo      fyne.ThemeColorName = "info"
	colorNameCreative  fyne.ThemeColorName = "creative"
	colorNameSystem    fyne.ThemeColorName = "system"
	colorNameAttention fyne.ThemeColorName = "attention"
)

var themeColors = map[fyne.ThemeColorName][]color.Color{
	colorNameInfo: { // light blue
		color.RGBA{0, 150, 255, 255},
		color.RGBA{77, 181, 255, 255},
	},
	colorNameCreative: { // purple
		color.RGBA{156, 39, 176, 255},
		color.RGBA{206, 147, 216, 255},
	},
	colorNameSystem: { // teal
		color.RGBA{0, 150, 136, 255},
		color.RGBA{77, 182, 172, 255},
	},
	colorNameAttention: { // Amber
		color.RGBA{255, 193, 7, 255},
		color.RGBA{255, 213, 79, 255},
	},
}

// customTheme represents a custom Fyne theme.
// It adds colors to the default theme
// and allows setting a fixed theme variant.
type customTheme struct {
	defaultTheme fyne.Theme
	mode         settings.ColorTheme
}

// newCustomTheme returns a new custom theme.
// Must be called after the app has started.
func newCustomTheme(mode settings.ColorTheme) *customTheme {
	th := &customTheme{
		defaultTheme: fyne.CurrentApp().Settings().Theme(),
		mode:         mode,
	}
	return th
}

func (th customTheme) Color(c fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch th.mode {
	case settings.Dark:
		v = theme.VariantDark
	case settings.Light:
		v = theme.VariantLight
	}
	switch c {
	case colorNameInfo, colorNameCreative, colorNameSystem, colorNameAttention:
		return themeColors[c][v]
	default:
		return th.defaultTheme.Color(c, v)
	}
}

func (th customTheme) Font(style fyne.TextStyle) fyne.Resource {
	return th.defaultTheme.Font(style)
}

func (th customTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return th.defaultTheme.Icon(n)
}

func (th customTheme) Size(s fyne.ThemeSizeName) float32 {
	return th.defaultTheme.Size(s)
}
