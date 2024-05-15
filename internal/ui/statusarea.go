package ui

import (
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// statusArea is the UI area showing the current status aka status bar.
type statusArea struct {
	content          *fyne.Container
	eveStatusTraffic *widget.Label
	eveStatusText    *widget.Label
	infoText         *widget.Label
	infoPB           *widget.ProgressBarInfinite
	ui               *ui
}

func (u *ui) newStatusArea() *statusArea {
	a := &statusArea{
		eveStatusTraffic: widget.NewLabel("•"),
		eveStatusText:    widget.NewLabel("Checking..."),
		infoText:         widget.NewLabel(""),
		infoPB:           widget.NewProgressBarInfinite(),
		ui:               u,
	}
	a.infoPB.Hide()
	c := container.NewHBox(
		container.NewHBox(a.infoText, a.infoPB),
		layout.NewSpacer(),
		container.NewHBox(a.eveStatusTraffic, a.eveStatusText))
	a.content = container.NewVBox(widget.NewSeparator(), c)
	return a
}

func (a *statusArea) StartUpdateTicker() {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for {
			var s string
			var i widget.Importance
			x, err := a.ui.service.FetchESIStatus()
			if err != nil {
				slog.Error("Failed to fetch ESI status", "err", err)
				i = widget.WarningImportance
				s = "ERROR"
			} else if !x.IsOnline {
				i = widget.DangerImportance
				s = "OFFLINE"
			} else {
				i = widget.SuccessImportance
				arg := message.NewPrinter(language.English)
				s = arg.Sprintf("%d players", x.PlayerCount)
			}
			a.eveStatusText.SetText(s)
			a.eveStatusTraffic.Importance = i
			a.eveStatusTraffic.Refresh()
			<-ticker.C
		}
	}()
}

func (s *statusArea) SetInfo(text string) {
	s.setInfo(text, widget.MediumImportance)
	s.infoPB.Stop()
	s.infoPB.Hide()
}

func (s *statusArea) SetInfoWithProgress(text string) {
	s.setInfo(text, widget.MediumImportance)
	s.infoPB.Start()
	s.infoPB.Show()
}

func (s *statusArea) SetError(text string) {
	s.setInfo(text, widget.DangerImportance)
	s.infoPB.Stop()
	s.infoPB.Hide()
}

func (s *statusArea) ClearInfo() {
	s.SetInfo("")
}

func (s *statusArea) setInfo(text string, importance widget.Importance) {
	s.infoText.Text = text
	s.infoText.Importance = importance
	s.infoText.Refresh()
}
