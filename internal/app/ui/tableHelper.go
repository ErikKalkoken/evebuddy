package ui

import (
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type headerDef struct {
	text     string
	maxChars int
}

func maxHeaderWidth(headers []headerDef) float32 {
	var m float32
	for _, h := range headers {
		l := widget.NewLabel(h.text)
		m = max(l.MinSize().Width, m)
	}
	return m
}

func makeRowLayout(headers []headerDef) fyne.Layout {
	return kxlayout.NewColumns(maxHeaderWidth(headers) + theme.Padding())
}

func makeListRowObject(headers []headerDef) fyne.CanvasObject {
	p := theme.Padding()
	rowLayout := makeRowLayout(headers)
	c := container.New(layout.NewCustomPaddedVBoxLayout(0))
	for _, h := range headers {
		row := container.New(rowLayout, widget.NewLabel(h.text), widget.NewLabel(""))
		bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
		bg.Hide()
		c.Add(container.NewStack(bg, row))
		c.Add(container.New(layout.NewCustomPaddedLayout(0, 0, 2*p, 2*p), widget.NewSeparator()))
	}
	return c
}
