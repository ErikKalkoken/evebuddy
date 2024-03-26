package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type statusBar struct {
	content *fyne.Container
	text    binding.String
}

func (s *statusBar) setText(text string) error {
	err := s.text.Set(text)
	return err
}

func (s *statusBar) clear() {
	s.setText("")
}

func (e *eveApp) newStatusBar() *statusBar {
	text := binding.NewString()
	label := widget.NewLabelWithData(text)
	content := container.NewVBox(widget.NewSeparator(), label)
	b := statusBar{
		content: content,
		text:    text,
	}
	return &b
}
