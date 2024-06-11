package ui

import (
	"bytes"
	"cmp"
	"context"
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
	"github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/golang/freetype/truetype"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

const (
	chartOtherThreshold = 0.05
)

type chartType uint

const (
	pieChart chartType = iota
	barChart
)

type wealthArea struct {
	content fyne.CanvasObject
	charts  *fyne.Container
	top     *widget.Label
	ui      *ui
}

func (u *ui) newWealthArea() *wealthArea {
	a := &wealthArea{
		ui:     u,
		charts: container.NewVBox(),
		top:    widget.NewLabel(""),
	}
	a.top.TextStyle.Bold = true
	a.content = container.NewBorder(
		container.NewVBox(a.top, widget.NewSeparator()), nil, nil, nil,
		container.NewScroll(a.charts))
	return a
}

type dataRow struct {
	label  string
	wallet float64
	assets float64
	total  float64
}

type myValue struct {
	label string
	value float64
}

// Fyne theme resources shared used in all charts
type myStyle struct {
	foregroundColor drawing.Color
	backgroundColor drawing.Color
	font            *truetype.Font
}

func (a *wealthArea) refresh() {
	data, err := a.compileData()
	if err != nil {
		panic(err)
	}
	style := compileThemeStyle()

	charactersData := make([]myValue, len(data))
	for i, r := range data {
		charactersData[i] = myValue{label: r.label, value: r.assets + r.wallet}
	}
	charactersChart := makeChart(pieChart, "Total Wealth By Character", charactersData, style)

	var totalWallet, totalAssets float64
	for _, r := range data {
		totalAssets += r.assets
		totalWallet += r.wallet
	}
	typesData := make([]myValue, 2)
	typesData[0] = myValue{label: "Assets", value: totalAssets}
	typesData[1] = myValue{label: "Wallet", value: totalWallet}
	typesChart := makeChart(pieChart, "Total Wealth By Type", typesData, style)

	assetsData := make([]myValue, len(data))
	for i, r := range data {
		assetsData[i] = myValue{label: r.label, value: r.assets}
	}
	assetsChart := makeChart(barChart, "Assets Value By Character", assetsData, style)

	walletData := make([]myValue, len(data))
	for i, r := range data {
		walletData[i] = myValue{label: r.label, value: r.wallet}
	}
	walletChart := makeChart(barChart, "Wallet Balance By Character", walletData, style)

	a.charts.RemoveAll()
	a.charts.Add(container.NewHBox(charactersChart, typesChart))
	a.charts.Add(assetsChart)
	a.charts.Add(walletChart)

	var total float64
	for _, r := range data {
		total += r.assets + r.wallet
	}
	a.top.SetText(fmt.Sprintf("Total: %s", humanize.Number(total, 1)))
}

func (a *wealthArea) compileData() ([]dataRow, error) {
	cc, err := a.ui.sv.Characters.ListCharacters(context.TODO())
	if err != nil {
		return nil, err
	}
	data := make([]dataRow, 0)
	for _, c := range cc {
		wallet := c.WalletBalance.Float64
		x, err := a.ui.sv.Characters.CharacterAssetTotalValue(c.ID)
		if err != nil {
			return nil, err
		}
		assets := x.Float64
		label := c.EveCharacter.Name
		r := dataRow{
			label:  label,
			assets: assets,
			wallet: wallet,
			total:  assets + wallet,
		}
		data = append(data, r)
	}
	slices.SortFunc(data, func(a, b dataRow) int {
		return cmp.Compare(a.total, b.total) * -1
	})
	return data, nil
}

func compileThemeStyle() myStyle {
	f := theme.DefaultTextFont().Content()
	font, err := truetype.Parse(f)
	if err != nil {
		panic(err)
	}
	style := myStyle{
		font:            font,
		foregroundColor: chartColor(theme.ForegroundColor()),
		backgroundColor: chartColor(theme.BackgroundColor()),
	}
	return style
}

func makeChart(ct chartType, title string, values []myValue, style myStyle) *fyne.Container {
	var content []byte
	var err error
	switch ct {
	case barChart:
		content, err = makeBarChart(values, style)
	case pieChart:
		content, err = makePieChart(values, style)
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
	t := widget.NewLabel(title)
	t.TextStyle.Bold = true
	c := container.NewPadded(container.NewBorder(
		container.NewHBox(t), nil, nil, nil,
		container.NewHBox(chart)))
	return c
}

func makeFileName(title string) string {
	c := cases.Title(language.English)
	fn := c.String(title)
	fn = strings.ReplaceAll(fn, " ", "")
	fn = fmt.Sprintf("%s.png", fn)
	return fn
}

func makePieChart(values []myValue, style myStyle) ([]byte, error) {
	var total, other float64
	for _, r := range values {
		total += r.value
	}
	data2 := make([]myValue, 0)
	for _, r := range values {
		if r.value/total < chartOtherThreshold {
			other += r.value
			continue
		}
		data2 = append(data2, r)
	}
	if other > 0 {
		data2 = append(data2, myValue{label: "Other", value: other})
	}
	chartValues := make([]chart.Value, 0)
	for _, r := range data2 {
		o := chart.Value{
			Label: fmt.Sprintf("%s %s", r.label, humanize.Number(r.value, 1)),
			Value: r.value,
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
			FontColor:   style.foregroundColor,
			StrokeColor: style.backgroundColor,
		},
		Font:   style.font,
		Values: chartValues,
	}
	var buf bytes.Buffer
	if err := pie.Render(chart.PNG, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func chartColor(c color.Color) drawing.Color {
	r, g, b, a := c.RGBA()
	return drawing.Color{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
}

func makeBarChart(data []myValue, style myStyle) ([]byte, error) {
	bars := make([]chart.Value, len(data))
	for i, r := range data {
		bars[i] = chart.Value{
			Label: r.label,
			Value: r.value,
		}
	}
	barChart := chart.BarChart{
		Background: chart.Style{
			FillColor: chart.ColorTransparent,
		},
		Canvas: chart.Style{
			FillColor: chart.ColorTransparent,
		},
		Font:   style.font,
		Width:  1024,
		Height: 512,
		XAxis: chart.Style{
			Hidden:    false,
			FontColor: style.foregroundColor,
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Hidden:    false,
				FontColor: style.foregroundColor,
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
