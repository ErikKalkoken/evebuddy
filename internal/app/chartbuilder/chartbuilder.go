// Package chartbuilder provides a chart builder for rending themed Fyne charts.
package chartbuilder

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"image/color"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app/humanize"
	"github.com/golang/freetype/truetype"
	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ChartType uint

const (
	Pie ChartType = iota
	Bar
)

const (
	chartOtherThreshold = 0.05
	defaultWidth        = 300
	defaultHeight       = 300
	chartPadding        = 0.05 // per cent
	defaultFontSize     = 10
)

type Value struct {
	Label string
	Value float64
}

var errInsufficientData = errors.New("insufficient data")

// CharBuilder renders themed Fyne charts.
type ChartBuilder struct {
	// Parameters must be set before calling Render to customize.
	ForegroundColor color.Color
	BackgroundColor color.Color
	Font            *truetype.Font
	FontSize        float64

	window fyne.Window
}

// New returns a new ChartBuilder object.
//
// The builder can be customized by setting it's exported fields.
func New(window fyne.Window) ChartBuilder {
	cb := ChartBuilder{
		ForegroundColor: color.Black,
		BackgroundColor: color.White,
		FontSize:        defaultFontSize,
		window:          window,
	}
	return cb
}

// Render returns a rendered chart in a Fyne container.
func (cb ChartBuilder) Render(ct ChartType, size fyne.Size, title string, values []Value) *fyne.Container {
	chart, err := cb.render(ct, size, title, values)
	if err != nil {
		var t string
		var i widget.Importance
		if err == errInsufficientData {
			t = "Insufficient data"
			i = widget.LowImportance
		} else {
			slog.Error("Failed to generate chart", "title", title, "err", err)
			t = fmt.Sprintf("Failed to generate chart: %s", humanize.Error(err))
			i = widget.DangerImportance
		}
		l := widget.NewLabel(t)
		l.Importance = i
		chart = container.NewCenter(l)
	}
	label := widget.NewLabel(title)
	label.TextStyle.Bold = true
	c := container.NewPadded(container.NewBorder(
		container.NewHBox(label), nil, nil, nil,
		container.NewHBox(chart)))
	return c
}

func (cb ChartBuilder) foregroundColor() drawing.Color {
	return chartColor(cb.ForegroundColor)
}

func (cb ChartBuilder) backgroundColor() drawing.Color {
	return chartColor(cb.BackgroundColor)
}

func (cb ChartBuilder) render(ct ChartType, size fyne.Size, title string, values []Value) (fyne.CanvasObject, error) {
	if len(values) < 2 {
		return nil, errInsufficientData
	}
	max := slices.MaxFunc(values, func(a, b Value) int {
		return cmp.Compare(a.Value, b.Value)
	})
	if max.Value == 0 {
		return nil, errInsufficientData
	}
	slices.SortFunc(values, func(a Value, b Value) int {
		return cmp.Compare(a.Label, b.Label)
	})
	pixelW, pixelH := imageSize(cb.window, size)
	var content []byte
	var err error
	switch ct {
	case Bar:
		content, err = cb.makeBarChart(pixelW, pixelH, values)
	case Pie:
		content, err = cb.makePieChart(pixelW, pixelH, values)
	}
	if err != nil {
		return nil, err
	}
	fn := makeFileName(title)
	r := fyne.NewStaticResource(fn, content)
	chart := canvas.NewImageFromResource(r)
	chart.SetMinSize(size)
	chart.FillMode = canvas.ImageFillContain
	return chart, nil
}

func makeFileName(title string) string {
	c := cases.Title(language.English)
	fn := c.String(title)
	fn = strings.ReplaceAll(fn, " ", "")
	fn = fmt.Sprintf("%s.png", fn)
	return fn
}

// imageSize returns the size of a chart image in pixel.
func imageSize(w fyne.Window, s fyne.Size) (int, int) {
	if w == nil {
		return 1, 1
	}
	f := w.Canvas().Scale()
	return int(s.Width * f), int(s.Height * f)
}

// func fontSize(w fyne.Window, size float32) float64 {
// 	if w == nil {
// 		return float64(size)
// 	}
// 	f := w.Canvas().Scale()
// 	return float64(size / f)
// }

func (cb ChartBuilder) makePieChart(width, height int, values []Value) ([]byte, error) {
	var total, other float64
	for _, r := range values {
		total += r.Value
	}
	data2 := make([]Value, 0)
	for _, r := range values {
		if r.Value/total < chartOtherThreshold {
			other += r.Value
			continue
		}
		data2 = append(data2, r)
	}
	if other > 0 {
		data2 = append(data2, Value{Label: "Other", Value: other})
	}
	chartValues := make([]chart.Value, 0)
	for _, r := range data2 {
		o := chart.Value{
			Label: fmt.Sprintf("%s %s", r.Label, humanize.Number(r.Value, 1)),
			Value: r.Value,
		}
		chartValues = append(chartValues, o)
	}
	pie := chart.PieChart{
		Width:  width,
		Height: height,
		Background: chart.Style{
			FillColor: chart.ColorTransparent,
			Padding: chart.Box{
				Top:    int(chartPadding * float32(height)),
				Bottom: int(chartPadding * float32(height)),
			},
		},
		Canvas: chart.Style{
			FillColor: chart.ColorTransparent,
			FontSize:  cb.FontSize,
		},
		SliceStyle: chart.Style{
			FontColor:   cb.foregroundColor(),
			StrokeColor: cb.backgroundColor(),
			FontSize:    cb.FontSize,
		},
		Values: chartValues,
	}
	if cb.Font != nil {
		pie.Font = cb.Font
	}
	var buf bytes.Buffer
	if err := pie.Render(chart.PNG, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (cb ChartBuilder) makeBarChart(width, height int, data []Value) ([]byte, error) {
	bars := make([]chart.Value, len(data))
	for i, r := range data {
		bars[i] = chart.Value{
			Label: r.Label,
			Value: r.Value,
		}
	}

	// fs := fontSize(cb.window, cb.FontSize)
	barChart := chart.BarChart{
		Background: chart.Style{
			FillColor: chart.ColorTransparent,
		},
		Canvas: chart.Style{
			FillColor: chart.ColorTransparent,
			FontSize:  cb.FontSize,
		},
		Width:  width,
		Height: height,
		XAxis: chart.Style{
			Hidden:              false,
			FontColor:           cb.foregroundColor(),
			FontSize:            cb.FontSize,
			TextRotationDegrees: 90,
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Hidden:    false,
				FontColor: cb.foregroundColor(),
				FontSize:  cb.FontSize,
			},
			ValueFormatter: numericValueFormatter,
		},
		Bars: bars,
	}
	if cb.Font != nil {
		barChart.Font = cb.Font
	}
	var buf bytes.Buffer
	if err := barChart.Render(chart.PNG, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func numericValueFormatter(v interface{}) string {
	x := v.(float64)
	return humanize.Number(x, 1)
}

func chartColor(c color.Color) drawing.Color {
	r, g, b, a := c.RGBA()
	return drawing.Color{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
}
