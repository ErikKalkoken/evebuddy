package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
)

const (
	ColorNameInfo      fyne.ThemeColorName = "info"
	ColorNameCreative  fyne.ThemeColorName = "creative"
	ColorNameSystem    fyne.ThemeColorName = "system"
	ColorNameAttention fyne.ThemeColorName = "attention"
)

var themeColors = map[fyne.ThemeColorName][]color.Color{
	ColorNameInfo: { // light blue
		color.RGBA{0, 150, 255, 255},
		color.RGBA{77, 181, 255, 255},
	},
	ColorNameCreative: { // purple
		color.RGBA{156, 39, 176, 255},
		color.RGBA{206, 147, 216, 255},
	},
	ColorNameSystem: { // teal
		color.RGBA{0, 150, 136, 255},
		color.RGBA{77, 182, 172, 255},
	},
	ColorNameAttention: { // Amber
		color.RGBA{255, 193, 7, 255},
		color.RGBA{255, 213, 79, 255},
	},
}

// Theme represents a custom Fyne theme.
// It adds colors to the default theme
// and allows setting a fixed theme variant.
type Theme struct {
	defaultTheme fyne.Theme
	mode         settings.ColorTheme
}

// New returns a new custom theme.
// Must be called after the app has started.
func New(defaultTheme fyne.Theme, mode settings.ColorTheme) *Theme {
	th := &Theme{
		defaultTheme: defaultTheme,
		mode:         mode,
	}
	return th
}

func (th Theme) Color(c fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch th.mode {
	case settings.Dark:
		v = theme.VariantDark
	case settings.Light:
		v = theme.VariantLight
	}
	switch c {
	case ColorNameInfo, ColorNameCreative, ColorNameSystem, ColorNameAttention:
		return themeColors[c][v]
	default:
		return th.defaultTheme.Color(c, v)
	}
}

func (th Theme) Font(style fyne.TextStyle) fyne.Resource {
	return th.defaultTheme.Font(style)
}

func (th Theme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return th.defaultTheme.Icon(n)
}

func (th Theme) Size(s fyne.ThemeSizeName) float32 {
	return th.defaultTheme.Size(s)
}
