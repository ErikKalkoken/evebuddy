package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type statusBar struct {
	content *fyne.Container
	label   *widget.Label
}

func (s *statusBar) update(text string) {
	s.label.SetText(text)
}

func (e *esiApp) newStatusBar() *statusBar {
	l := widget.NewLabel("")
	c := container.NewVBox(widget.NewSeparator(), l)
	s := statusBar{
		content: c,
		label:   l,
	}
	return &s
}
