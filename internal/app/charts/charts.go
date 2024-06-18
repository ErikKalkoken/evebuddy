// Package charts provides a chart builder for rending themed Fyne charts.
package charts

import (
	"bytes"
	"cmp"
	"fmt"
	"image/color"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
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
)

type Value struct {
	Label string
	Value float64
}

// CharBuilder renders themed Fyne charts.
type ChartBuilder struct {
	foregroundColor drawing.Color
	backgroundColor drawing.Color
	font            *truetype.Font
}

func NewChartBuilder() ChartBuilder {
	f := theme.DefaultTextFont().Content()
	font, err := truetype.Parse(f)
	if err != nil {
		panic(err)
	}
	cb := ChartBuilder{
		font:            font,
		foregroundColor: chartColor(theme.ForegroundColor()),
		backgroundColor: chartColor(theme.BackgroundColor()),
	}
	return cb
}

// Render returns a rendered chart in a Fyne container.
func (cb ChartBuilder) Render(ct ChartType, title string, values []Value) *fyne.Container {
	chart := cb.render(ct, title, values)
	label := widget.NewLabel(title)
	label.TextStyle.Bold = true
	c := container.NewPadded(container.NewBorder(
		container.NewHBox(label), nil, nil, nil,
		container.NewHBox(chart)))
	return c
}

func (cb ChartBuilder) render(ct ChartType, title string, values []Value) fyne.CanvasObject {
	max := slices.MaxFunc(values, func(a, b Value) int {
		return cmp.Compare(a.Value, b.Value)
	})
	if len(values) < 2 || max.Value == 0 {
		l := widget.NewLabel("Insufficient data")
		l.Importance = widget.LowImportance
		return container.NewCenter(l)
	}
	var content []byte
	var err error
	switch ct {
	case Bar:
		content, err = cb.makeBarChart(values)
	case Pie:
		content, err = cb.makePieChart(values)
	}
	if err != nil {
		slog.Error("Failed to generate chart", "title", title, "err", err)
		l := widget.NewLabel(fmt.Sprintf("Failed to generate chart: %s", humanize.Error(err)))
		l.Importance = widget.DangerImportance
		return container.NewCenter(l)
	}
	fn := makeFileName(title)
	r := fyne.NewStaticResource(fn, content)
	chart := canvas.NewImageFromResource(r)
	chart.FillMode = canvas.ImageFillOriginal
	return chart
}

func makeFileName(title string) string {
	c := cases.Title(language.English)
	fn := c.String(title)
	fn = strings.ReplaceAll(fn, " ", "")
	fn = fmt.Sprintf("%s.png", fn)
	return fn
}

func (cb ChartBuilder) makePieChart(values []Value) ([]byte, error) {
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
		Width:  512,
		Height: 512,
		Background: chart.Style{
			FillColor: chart.ColorTransparent,
			Padding: chart.Box{
				Top:    25,
				Bottom: 25,
			},
		},
		Canvas: chart.Style{
			FillColor: chart.ColorTransparent,
		},
		SliceStyle: chart.Style{
			FontColor:   cb.foregroundColor,
			StrokeColor: cb.backgroundColor,
		},
		Font:   cb.font,
		Values: chartValues,
	}
	var buf bytes.Buffer
	if err := pie.Render(chart.PNG, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (cb ChartBuilder) makeBarChart(data []Value) ([]byte, error) {
	bars := make([]chart.Value, len(data))
	for i, r := range data {
		bars[i] = chart.Value{
			Label: r.Label,
			Value: r.Value,
		}
	}
	barChart := chart.BarChart{
		Background: chart.Style{
			FillColor: chart.ColorTransparent,
		},
		Canvas: chart.Style{
			FillColor: chart.ColorTransparent,
		},
		Font:   cb.font,
		Width:  1024,
		Height: 512,
		XAxis: chart.Style{
			Hidden:    false,
			FontColor: cb.foregroundColor,
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Hidden:    false,
				FontColor: cb.foregroundColor,
			},
			ValueFormatter: numericValueFormatter,
		},
		Bars: bars,
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
