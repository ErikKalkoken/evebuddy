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

type myTheme struct {
	mode settings.ColorTheme
}

func (ct myTheme) Color(c fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch ct.mode {
	case settings.Dark:
		v = theme.VariantDark
	case settings.Light:
		v = theme.VariantLight
	}
	switch c {
	case colorNameInfo, colorNameCreative, colorNameSystem, colorNameAttention:
		return themeColors[c][v]
	default:
		return theme.DefaultTheme().Color(c, v)
	}
}

func (myTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (myTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (myTheme) Size(s fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(s)
}
