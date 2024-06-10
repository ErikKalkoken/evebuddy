package ui

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/wcharczuk/go-chart"
)

const (
	chartOtherThreshold = 0.05
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
		charts: container.NewStack(),
		top:    widget.NewLabel(""),
	}
	a.top.TextStyle.Bold = true
	a.content = container.NewBorder(container.NewVBox(a.top, widget.NewSeparator()), nil, nil, nil, a.charts)
	return a
}

func (a *wealthArea) refresh() {
	cc, err := a.ui.sv.Characters.ListCharacters(context.TODO())
	if err != nil {
		panic(err)
	}
	var total, other float64
	m := make(map[string]float64)
	for _, c := range cc {
		wallet := c.WalletBalance.Float64
		x, err := a.ui.sv.Characters.CharacterAssetTotalValue(c.ID)
		if err != nil {
			panic(err)
		}
		assets := x.Float64
		label := c.EveCharacter.Name
		v := wallet + assets
		m[label] = v
		total += v
	}
	for k, v := range m {
		if v/total < chartOtherThreshold {
			other += v
			delete(m, k)
		}
	}
	if other > 0 {
		m["Other"] = other
	}
	values := make([]chart.Value, 0)
	for k, v := range m {
		o := chart.Value{
			Label: fmt.Sprintf("%s %s", k, humanize.Number(v, 1)),
			Value: v,
		}
		values = append(values, o)
	}
	slices.SortFunc(values, func(a, b chart.Value) int {
		return cmp.Compare(a.Value, b.Value) * -1
	})
	pie := chart.PieChart{
		Title:        "Sexy chart",
		Width:        1024,
		Height:       1024,
		ColorPalette: chartColorPalette{},
		Values:       values,
	}
	var buf bytes.Buffer
	pie.Render(chart.PNG, &buf)

	r := fyne.NewStaticResource("chart.png", buf.Bytes())
	i := canvas.NewImageFromResource(r)
	i.FillMode = canvas.ImageFillContain
	i.SetMinSize(fyne.Size{Width: 512, Height: 512})
	c := container.NewStack(i)
	a.charts.RemoveAll()
	a.charts.Add(c)
	a.top.SetText(fmt.Sprintf("Total: %s", humanize.Number(total, 1)))
}
