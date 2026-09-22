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
	wealthArcInnerRadius  = 0.6
	wealthArcPadAngle     = 1.5
)

// wealthWalletSeriesColor, wealthContractsSeriesColor, wealthOrdersSeriesColor,
// wealthPurpleSeriesColor, and wealthTealSeriesColor match fyneline's default
// series colors.
var (
	wealthWalletSeriesColor    = color.NRGBA{R: 240, G: 135, B: 48, A: 255}
	wealthContractsSeriesColor = color.NRGBA{R: 47, G: 176, B: 117, A: 255}
	wealthOrdersSeriesColor    = color.NRGBA{R: 220, G: 72, B: 103, A: 255}
	wealthPurpleSeriesColor    = color.NRGBA{R: 139, G: 92, B: 246, A: 255}
	wealthTealSeriesColor      = color.NRGBA{R: 16, G: 164, B: 190, A: 255}
)

// wealthArcPalette mirrors fyneline's internal default series color cycle, so
// a manually-built legend can match arc-chart slice colors by index (index 0
// uses the theme primary color instead, see wealthSliceColor).
var wealthArcPalette = []color.Color{
	color.NRGBA{R: 62, G: 126, B: 247, A: 255},
	wealthWalletSeriesColor,
	wealthContractsSeriesColor,
	wealthOrdersSeriesColor,
	wealthPurpleSeriesColor,
	wealthTealSeriesColor,
}

// wealthSliceColor returns the color fyneline's ArcChart uses for the slice
// at index, so a manually-built legend can match it exactly.
func wealthSliceColor(w fyne.Widget, index int) color.Color {
	if index == 0 {
		return theme.ColorForWidget(theme.ColorNamePrimary, w)
	}
	return wealthArcPalette[index%len(wealthArcPalette)]
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
		legend = w.legend
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

// legendSwatch is a color swatch that tracks a theme-derived color.
type legendSwatch struct {
	rect    *canvas.Rectangle
	colorFn func() color.Color
}

func newLegendSwatch(colorFn func() color.Color) *legendSwatch {
	return &legendSwatch{rect: canvas.NewRectangle(colorFn()), colorFn: colorFn}
}

func (s *legendSwatch) object() fyne.CanvasObject {
	const swatchSize = 12
	return container.NewGridWrap(fyne.NewSize(swatchSize, swatchSize), s.rect)
}

func newLegendEntry(label string, swatch *legendSwatch) fyne.CanvasObject {
	// Match the text size fyneline uses for its axis labels.
	l := widget.NewLabel(label)
	l.SizeName = theme.SizeNameCaptionText
	return container.NewHBox(container.NewCenter(swatch.object()), container.NewCenter(l))
}

// seriesLegend wraps a row of legend entries (as built by [newLegendEntry]),
// wrapping onto multiple lines as needed, for use in a chartCard's legend slot.
type seriesLegend struct {
	widget.BaseWidget

	container *fyne.Container
}

// newSeriesLegend creates a legend from entries (as built by [newLegendEntry]).
func newSeriesLegend(entries ...fyne.CanvasObject) *seriesLegend {
	w := &seriesLegend{container: container.New(layout.NewRowWrapLayout(), entries...)}
	w.ExtendBaseWidget(w)
	return w
}

func (w *seriesLegend) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(w.container)
}

// SetEntries replaces the legend's entries (as built by [newLegendEntry]).
func (w *seriesLegend) SetEntries(entries ...fyne.CanvasObject) {
	w.container.Objects = entries
	w.container.Refresh()
}
