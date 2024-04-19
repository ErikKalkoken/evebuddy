package ui

import (
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// statusArea is the UI area showing the current status aka status bar.
type statusArea struct {
	content       *fyne.Container
	infoText      binding.String
	eveStatusText binding.String
	ui            *ui
}

func (u *ui) newStatusArea() *statusArea {
	infoText := binding.NewString()
	infoLabel := widget.NewLabelWithData(infoText)
	statusText := binding.NewString()
	statusLabel := widget.NewLabelWithData(statusText)
	c := container.NewHBox(infoLabel, layout.NewSpacer(), statusLabel)
	content := container.NewVBox(widget.NewSeparator(), c)
	b := statusArea{
		content:       content,
		infoText:      infoText,
		eveStatusText: statusText,
		ui:            u,
	}
	return &b
}

func (s *statusArea) setInfo(text string) error {
	err := s.infoText.Set(text)
	return err
}

// func (s *statusArea) clearInfo() error {
// 	err := s.info.Set("")
// 	return err
// }

func (s *statusArea) StartUpdateTicker() {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for {
			t, err := s.ui.service.FetchESIStatus()
			if err != nil {
				slog.Error(err.Error())
			} else {
				if err := s.eveStatusText.Set(t); err != nil {
					slog.Error(err.Error())
				}
			}
			<-ticker.C
		}
	}()
}
