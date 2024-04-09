package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// statusArea is the UI area showing the current status aka status bar.
type statusArea struct {
	content *fyne.Container
	info    binding.String
	status  binding.String
	ui      *ui
}

func (u *ui) newStatusArea() *statusArea {
	infoText := binding.NewString()
	infoLabel := widget.NewLabelWithData(infoText)
	statusText := binding.NewString()
	statusLabel := widget.NewLabelWithData(statusText)
	c := container.NewHBox(infoLabel, layout.NewSpacer(), statusLabel)
	content := container.NewVBox(widget.NewSeparator(), c)
	b := statusArea{
		content: content,
		info:    infoText,
		status:  statusText,
		ui:      u,
	}
	return &b
}

func (s *statusArea) setText(text string) error {
	err := s.info.Set(text)
	return err
}
