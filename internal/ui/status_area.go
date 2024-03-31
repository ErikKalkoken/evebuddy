package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

// statusArea is the UI area showing the current status aka status bar.
type statusArea struct {
	content *fyne.Container
	text    binding.String
}

func (u *ui) newStatusArea() *statusArea {
	text := binding.NewString()
	label := widget.NewLabelWithData(text)
	content := container.NewVBox(widget.NewSeparator(), label)
	b := statusArea{
		content: content,
		text:    text,
	}
	return &b
}

func (s *statusArea) setText(text string) error {
	err := s.text.Set(text)
	return err
}

func (s *statusArea) clear() {
	s.setText("")
}
