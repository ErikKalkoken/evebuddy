package ui

import (
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// statusArea is the UI area showing the current status aka status bar.
type statusArea struct {
	content   *fyne.Container
	info      *widget.Label
	eveStatus *widget.Label
	ui        *ui
}

func (u *ui) newStatusArea() *statusArea {
	infoLabel := widget.NewLabel("")
	statusLabel := widget.NewLabel("")
	c := container.NewHBox(infoLabel, layout.NewSpacer(), statusLabel)
	content := container.NewVBox(widget.NewSeparator(), c)
	b := statusArea{
		content:   content,
		info:      infoLabel,
		eveStatus: statusLabel,
		ui:        u,
	}
	return &b
}

func (s *statusArea) setInfo(text string) {
	s.info.SetText(text)
}

func (s *statusArea) clearInfo() {
	s.setInfo("")
}

func (s *statusArea) StartUpdateTicker() {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for {
			t, err := s.ui.service.FetchESIStatus()
			if err != nil {
				slog.Error(err.Error())
			} else {
				s.eveStatus.SetText(t)
			}
			<-ticker.C
		}
	}()
}
