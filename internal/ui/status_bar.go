package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type statusBar struct {
	content *fyne.Container
	label   binding.String
}

func (s *statusBar) update(text string) {
	err := s.label.Set(text)
	if err != nil {
		log.Printf("Failed to set status label: %v", err)
	}
}

func (e *esiApp) newStatusBar() *statusBar {
	label := binding.NewString()
	l := widget.NewLabelWithData(label)
	c := container.NewVBox(widget.NewSeparator(), l)
	s := statusBar{
		content: c,
		label:   label,
	}
	return &s
}
