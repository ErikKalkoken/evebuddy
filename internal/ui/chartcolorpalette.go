package ui

import (
	"fyne.io/fyne/v2/theme"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

type chartColorPalette struct{}

func (dp chartColorPalette) BackgroundColor() drawing.Color {
	return chart.ColorTransparent
}

func (dp chartColorPalette) BackgroundStrokeColor() drawing.Color {
	return chart.DefaultBackgroundStrokeColor
}

func (dp chartColorPalette) CanvasColor() drawing.Color {
	return chart.ColorTransparent
}

func (dp chartColorPalette) CanvasStrokeColor() drawing.Color {
	return chart.DefaultCanvasStrokeColor
}

func (dp chartColorPalette) AxisStrokeColor() drawing.Color {
	return chart.ColorTransparent
}

func (dp chartColorPalette) TextColor() drawing.Color {
	c := theme.ForegroundColor()
	r, g, b, a := c.RGBA()
	x := drawing.Color{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
	return x
}

func (dp chartColorPalette) GetSeriesColor(index int) drawing.Color {
	finalIndex := index % len(DefaultColors)
	return DefaultColors[finalIndex]
}

var DefaultColors = []drawing.Color{
	chart.ColorBlue,
	chart.ColorGreen,
	chart.ColorRed,
	chart.ColorCyan,
	chart.ColorOrange,
}
