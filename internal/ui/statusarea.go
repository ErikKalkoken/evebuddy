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
	content          *fyne.Container
	Info             *statusInfo
	eveStatusTraffic *widget.Label
	eveStatusText    *widget.Label
	ui               *ui
}

func (u *ui) newStatusArea() *statusArea {
	info := newStatusInfo()
	statusTraffic := widget.NewLabel("•")
	statusText := widget.NewLabel("")
	c := container.NewHBox(info.content, layout.NewSpacer(), container.NewHBox(statusTraffic, statusText))
	content := container.NewVBox(widget.NewSeparator(), c)
	b := statusArea{
		content:          content,
		Info:             info,
		eveStatusTraffic: statusTraffic,
		eveStatusText:    statusText,
		ui:               u,
	}
	return &b
}

func (a *statusArea) StartUpdateTicker() {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for {
			var i widget.Importance
			t, err := a.ui.service.FetchESIStatus()
			if err != nil {
				slog.Error(err.Error())
				i = widget.WarningImportance
			} else {
				if t == "OFFLINE" {
					i = widget.SuccessImportance
				} else {
					i = widget.SuccessImportance
				}
			}
			a.eveStatusText.SetText(t)
			a.eveStatusTraffic.Importance = i
			a.eveStatusTraffic.Refresh()
			<-ticker.C
		}
	}()
}

type statusInfo struct {
	content *fyne.Container
}

func newStatusInfo() *statusInfo {
	return &statusInfo{content: container.NewHBox()}
}

func (s *statusInfo) Set(text string) {
	s.content.RemoveAll()
	s.content.Add(widget.NewLabel(text))
}

func (s *statusInfo) SetWithProgress(text string) {
	s.Set(text)
	s.content.Add(widget.NewProgressBarInfinite())
}

func (s *statusInfo) SetError(text string) {
	s.content.RemoveAll()
	l := widget.NewLabel(text)
	l.Importance = widget.DangerImportance
	s.content.Add(l)
}

func (s *statusInfo) Clear() {
	s.Set("")
}
