package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type statusBar struct {
	content *fyne.Container
	label   *widget.Label
}

func (s *statusBar) setText(format string, a ...any) {
	s.label.SetText(fmt.Sprintf(format, a...))
}

func (s *statusBar) clear() {
	s.setText("")
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
