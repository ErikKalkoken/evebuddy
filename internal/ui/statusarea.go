package ui

import (
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// statusArea is the UI area showing the current status aka status bar.
type statusArea struct {
	content                  *fyne.Container
	eveClock                 *widget.Label
	eveStatusTrafficIcon     *widget.Icon
	eveStatusTrafficResource fyne.Resource
	eveStatusText            *widget.Label
	infoText                 *widget.Label
	infoPB                   *widget.ProgressBarInfinite
	ui                       *ui
}

func (u *ui) newStatusArea() *statusArea {
	a := &statusArea{
		eveClock:      widget.NewLabel(""),
		eveStatusText: widget.NewLabel("Checking..."),
		infoText:      widget.NewLabel(""),
		infoPB:        widget.NewProgressBarInfinite(),
		ui:            u,
	}
	a.infoPB.Hide()
	a.eveStatusTrafficResource = theme.MediaRecordIcon()
	a.eveStatusTrafficIcon = widget.NewIcon(theme.NewDisabledResource(a.eveStatusTrafficResource))
	c := container.NewHBox(
		container.NewHBox(a.infoText, a.infoPB),
		layout.NewSpacer(),
		widget.NewSeparator(),
		container.NewHBox(
			a.eveClock, layout.NewSpacer(), widget.NewSeparator(), a.eveStatusTrafficIcon, a.eveStatusText))
	a.content = container.NewVBox(widget.NewSeparator(), c)
	return a
}

func (a *statusArea) StartUpdateTicker() {
	clockTicker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			t := time.Now().UTC()
			a.eveClock.SetText(t.Format("15:04"))
			<-clockTicker.C
		}
	}()
	statusTicker := time.NewTicker(60 * time.Second)
	go func() {
		for {
			var s string
			var r fyne.Resource
			x, err := a.ui.service.FetchESIStatus()
			if err != nil {
				slog.Error("Failed to fetch ESI status", "err", err)
				r = theme.NewErrorThemedResource(a.eveStatusTrafficResource)
				s = "ERROR"
			} else if !x.IsOnline {
				r = theme.NewWarningThemedResource(a.eveStatusTrafficResource)
				s = "OFFLINE"
			} else {
				r = theme.NewSuccessThemedResource(a.eveStatusTrafficResource)
				arg := message.NewPrinter(language.English)
				s = arg.Sprintf("%d players", x.PlayerCount)
			}
			a.eveStatusText.SetText(s)
			a.eveStatusTrafficIcon.SetResource(r)
			<-statusTicker.C
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
