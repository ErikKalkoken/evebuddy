package screens

import (
	"fmt"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/nathabonfim59/fyneline"
)

const (
	wealthArcCornerRadius = 8
	wealthArcInnerRadius  = 0.58
	wealthArcPadAngle     = 3
	swatchSize            = 12
)

// wealthBlueColor etc. are the building blocks of wealthPalette below; the
// first 6 seed from fyneline's own default series colors, extended with more
// to cover charts with more segments than fyneline's default palette has.
var (
	wealthBlueColor    = color.NRGBA{R: 62, G: 126, B: 247, A: 255}
	wealthOrangeColor  = color.NRGBA{R: 240, G: 135, B: 48, A: 255}
	wealthGreenColor   = color.NRGBA{R: 47, G: 176, B: 117, A: 255}
	wealthRedColor     = color.NRGBA{R: 220, G: 72, B: 103, A: 255}
	wealthPurpleColor  = color.NRGBA{R: 139, G: 92, B: 246, A: 255}
	wealthTealColor    = color.NRGBA{R: 16, G: 164, B: 190, A: 255}
	wealthYellowColor  = color.NRGBA{R: 234, G: 179, B: 8, A: 255}
	wealthLimeColor    = color.NRGBA{R: 132, G: 204, B: 22, A: 255}
	wealthFuchsiaColor = color.NRGBA{R: 217, G: 70, B: 239, A: 255}
	wealthIndigoColor  = color.NRGBA{R: 99, G: 102, B: 241, A: 255}
	wealthBrownColor   = color.NRGBA{R: 161, G: 98, B: 7, A: 255}
	wealthGrayColor    = color.NRGBA{R: 100, G: 116, B: 139, A: 255}
)

// wealthPalette is the palette shared by every wealth arc chart and its
// legend, applied via wealthArcStyle so both always agree on segment colors.
var wealthPalette = []color.Color{
	wealthBlueColor,
	wealthOrangeColor,
	wealthGreenColor,
	wealthRedColor,
	wealthPurpleColor,
	wealthTealColor,
	wealthYellowColor,
	wealthLimeColor,
	wealthFuchsiaColor,
	wealthIndigoColor,
	wealthBrownColor,
	wealthGrayColor,
}

// wealthArcColor returns wealthPalette's color for a slice index, cycling
// if index exceeds the palette length.
var wealthArcColor = fyneline.Palette(wealthPalette...)

// wealthArcStyle assigns each arc chart segment its wealthPalette color.
func wealthArcStyle(_ namedValue, index int) fyneline.ArcStyle {
	return fyneline.ArcStyle{Fill: fyneline.FillStyle{Color: wealthArcColor(index), Opacity: 1}}
}

// namedValue is a single category/value pair used by the charts.
type namedValue struct {
	name  string
	value float64
}

// newChartTitleLabel returns a bold label for a chart card's title.
func newChartTitleLabel() *widget.Label {
	return widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

// configureArcChart applies this screen's shared pie/doughnut styling.
func configureArcChart(chart *fyneline.ArcChart[namedValue]) {
	chart.SetLabels(true)
	chart.SetInnerRadius(wealthArcInnerRadius)
	chart.SetPadAngle(wealthArcPadAngle)
	chart.SetCornerRadius(wealthArcCornerRadius)
	chart.SetStyle(wealthArcStyle)
}

// wealthAxisValueFormatter formats a value-axis tick to 1 decimal.
func wealthAxisValueFormatter(v float64) string { return fmt.Sprintf("%.1f", v) }

// niceAxisBounds picks an axis max and tick count so ticks fall on round
// step boundaries and the last tick sits close to value, instead of value
// possibly landing well short of a coarsely-rounded max (e.g. 74 -> 100).
func niceAxisBounds(value float64, targetIntervals int) (axisMax float64, tickCount int) {
	if value <= 0 {
		return 1, 2
	}
	step := niceStep(value / float64(max(targetIntervals, 1)))
	axisMax = step * math.Ceil(value/step)
	return axisMax, int(math.Round(axisMax/step)) + 1
}

// niceStep rounds value up to the nearest "nice" 1-2-5-10 number at its
// order of magnitude.
func niceStep(value float64) float64 {
	magnitude := math.Pow(10, math.Floor(math.Log10(value)))
	normalized := value / magnitude
	var niceFraction float64
	switch {
	case normalized <= 1:
		niceFraction = 1
	case normalized <= 2:
		niceFraction = 2
	case normalized <= 5:
		niceFraction = 5
	default:
		niceFraction = 10
	}
	return niceFraction * magnitude
}

// chartCard wraps a chart with a title, a legend, and a themed grey
// backdrop, and adds theming support.
type chartCard struct {
	widget.BaseWidget

	title   *widget.Label
	legend  *seriesLegend
	chart   fyne.CanvasObject
	bg      *canvas.Rectangle
	variant fyne.ThemeVariant
}

// newChartCard creates a chart card; applyTheme reapplies colors fyneline
// itself won't re-derive, e.g. WithFill fills.
func newChartCard(title *widget.Label, legend *seriesLegend, chart fyne.CanvasObject) *chartCard {
	w := &chartCard{
		title:  title,
		legend: legend,
		chart:  chart,
		bg:     canvas.NewRectangle(color.Transparent),
	}
	w.ExtendBaseWidget(w)
	w.bg.CornerRadius = theme.Size(theme.SizeNameCardRadius)
	w.bg.FillColor = theme.Color(theme.ColorNameInputBackground)
	return w
}

func (w *chartCard) CreateRenderer() fyne.WidgetRenderer {
	// w.legend is a typed *seriesLegend; passing a nil one straight into
	// NewBorder's fyne.CanvasObject param would box a non-nil interface
	// around a nil pointer, so nil it out explicitly instead.
	var legend fyne.CanvasObject
	if w.legend != nil {
		p := theme.Padding()
		legend = container.New(layout.NewCustomPaddedLayout(0, 0, 2*p, 0), w.legend)
	}
	content := container.NewBorder(w.title, legend, nil, nil, w.chart)
	return widget.NewSimpleRenderer(container.NewStack(w.bg, container.NewPadded(content)))
}

func (w *chartCard) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(theme.ColorNameInputBackground, v)
	w.bg.Refresh()
	w.BaseWidget.Refresh()
}

// legendEntry is a single legend row: a color swatch plus a label.
type legendEntry struct {
	widget.BaseWidget

	rect  *canvas.Rectangle
	label *widget.Label
}

func newLegendEntry(text string, c color.Color) *legendEntry {
	label := widget.NewLabel(text)
	label.SizeName = theme.SizeNameCaptionText // match fyneline's axis label size
	w := &legendEntry{rect: canvas.NewRectangle(c), label: label}
	w.ExtendBaseWidget(w)
	return w
}

func (w *legendEntry) CreateRenderer() fyne.WidgetRenderer {
	// p := theme.Padding()
	swatch := container.NewGridWrap(fyne.NewSize(swatchSize, swatchSize), w.rect)
	c := container.New(layout.NewCustomPaddedHBoxLayout(0), container.NewCenter(swatch), container.NewCenter(w.label))
	return widget.NewSimpleRenderer(c)
}

// seriesLegend wraps a row of [legendEntry] items, wrapping onto multiple
// lines as needed, for use in a chartCard's legend slot.
type seriesLegend struct {
	widget.BaseWidget

	container *fyne.Container
}

// newSeriesLegend creates a legend from entries.
func newSeriesLegend(entries ...*legendEntry) *seriesLegend {
	p := theme.Padding()
	w := &seriesLegend{container: container.New(
		layout.NewRowWrapLayoutWithCustomPadding(p, -2*p),
		legendEntryObjects(entries)...,
	)}
	w.ExtendBaseWidget(w)
	return w
}

func (w *seriesLegend) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewPadded(w.container))
}

// SetEntries replaces the legend's entries.
func (w *seriesLegend) SetEntries(entries ...*legendEntry) {
	w.container.Objects = legendEntryObjects(entries)
	w.container.Refresh()
}

func legendEntryObjects(entries []*legendEntry) []fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, len(entries))
	for i, e := range entries {
		objects[i] = e
	}
	return objects
}
