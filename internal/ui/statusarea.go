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
	Info      *statusInfo
	eveStatus *widget.Label
	ui        *ui
}

func (u *ui) newStatusArea() *statusArea {
	info := newStatusInfo()
	statusLabel := widget.NewLabel("")
	c := container.NewHBox(info.content, layout.NewSpacer(), statusLabel)
	content := container.NewVBox(widget.NewSeparator(), c)
	b := statusArea{
		content:   content,
		Info:      info,
		eveStatus: statusLabel,
		ui:        u,
	}
	return &b
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
