package ui

import (
	"context"
	"log/slog"
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/ErikKalkoken/evebuddy/internal/app/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/ErikKalkoken/evebuddy/internal/github"
)

const (
	characterUpdateStatusTicker = 2 * time.Second
	clockUpdateTicker           = 2 * time.Second
	esiStatusUpdateTicker       = 60 * time.Second
	githubOwner                 = "ErikKalkoken"
	githubRepo                  = "evebuddy"
	websiteURL                  = "https://github.com/ErikKalkoken/evebuddy"
)

type eveStatus uint

const (
	eveStatusUnknown eveStatus = iota
	eveStatusOnline
	eveStatusOffline
	eveStatusError
)

// statusBarArea is the UI area showing the current status aka status bar.
type statusBarArea struct {
	content            *fyne.Container
	eveClock           binding.String
	eveStatus          *widgets.StatusBarItem
	eveStatusError     string
	infoText           *widget.Label
	updateNotification *fyne.Container
	updateStatus       *widgets.StatusBarItem
	ui                 *ui
}

func (u *ui) newStatusBarArea() *statusBarArea {
	a := &statusBarArea{
		infoText:           widget.NewLabel(""),
		eveClock:           binding.NewString(),
		updateNotification: container.NewHBox(),
		ui:                 u,
	}
	a.updateStatus = widgets.NewStatusBarItem(nil, "?", func() {
		u.showStatusWindow()
	})
	a.eveStatus = widgets.NewStatusBarItem(theme.MediaRecordIcon(), "?", a.showDetail)

	a.eveClock.Set("?")
	clock := widget.NewLabelWithData(a.eveClock)
	a.content = container.NewVBox(widget.NewSeparator(), container.NewHBox(
		a.infoText,
		layout.NewSpacer(),
		a.updateNotification,
		widget.NewSeparator(),
		a.updateStatus,
		widget.NewSeparator(),
		clock,
		widget.NewSeparator(),
		a.eveStatus,
	))
	return a
}

func (a *statusBarArea) showDetail() {
	var i widget.Importance
	var text string
	if a.eveStatusError == "" {
		text = "No error detected"
		i = widget.MediumImportance
	} else {
		text = a.eveStatusError
		i = widget.DangerImportance
	}
	lb := widget.NewLabel(text)
	lb.Wrapping = fyne.TextWrapWord
	lb.Importance = i
	d := dialog.NewCustom("ESI status", "OK", lb, a.ui.window)
	d.Show()
	d.Resize(fyne.Size{Width: 400, Height: 200})
}

func (a *statusBarArea) StartUpdateTicker() {
	clockTicker := time.NewTicker(clockUpdateTicker)
	go func() {
		for {
			t := time.Now().UTC().Format("15:04")
			a.eveClock.Set(t)
			<-clockTicker.C
		}
	}()
	if a.ui.isOffline {
		a.setEveStatus(eveStatusOffline, "OFFLINE", "Offline mode")
		a.refreshUpdateStatus()
		return
	}
	updateTicker := time.NewTicker(characterUpdateStatusTicker)
	go func() {
		for {
			a.refreshUpdateStatus()
			<-updateTicker.C
		}
	}()
	esiStatusTicker := time.NewTicker(esiStatusUpdateTicker)
	go func() {
		for {
			x, err := a.ui.ESIStatusService.Fetch(context.TODO())
			var t, errorMessage string
			var s eveStatus
			if err != nil {
				slog.Error("Failed to fetch ESI status", "err", err)
				errorMessage = humanize.Error(err)
				s = eveStatusError
				t = "ERROR"
			} else if !x.IsOK() {
				errorMessage = x.ErrorMessage
				s = eveStatusOffline
				t = "OFFLINE"
			} else {
				arg := message.NewPrinter(language.English)
				t = arg.Sprintf("%d players", x.PlayerCount)
				s = eveStatusOnline
			}
			a.setEveStatus(s, t, errorMessage)
			<-esiStatusTicker.C
		}
	}()
	go func() {
		current := a.ui.fyneApp.Metadata().Version
		_, isNewer, err := github.AvailableUpdate(githubOwner, githubRepo, current)
		if err != nil {
			slog.Error("Failed to fetch latest version from github", "err", err)
			return
		}
		if !isNewer {
			return
		}
		x, _ := url.Parse(websiteURL + "/releases")
		l := widget.NewHyperlink("Update available", x)
		a.updateNotification.Add(widget.NewSeparator())
		a.updateNotification.Add(l)
	}()
}

func (a *statusBarArea) refreshUpdateStatus() {
	x := a.ui.StatusCacheService.Summary()
	a.updateStatus.SetTextAndImportance(x.Display(), status2widgetImportance(x.Status()))
}

func (a *statusBarArea) setEveStatus(status eveStatus, title, errorMessage string) {
	a.eveStatusError = errorMessage
	r1 := theme.MediaRecordIcon()
	var r2 fyne.Resource
	switch status {
	case eveStatusOnline:
		r2 = theme.NewSuccessThemedResource(r1)
	case eveStatusError:
		r2 = theme.NewErrorThemedResource(r1)
	case eveStatusOffline:
		r2 = theme.NewWarningThemedResource(r1)
	case eveStatusUnknown:
		r2 = theme.NewDisabledResource(r1)
	}
	a.eveStatus.SetResource(r2)
	a.eveStatus.SetText(title)
}

func (s *statusBarArea) SetInfo(text string) {
	s.setInfo(text, widget.MediumImportance)
}

func (s *statusBarArea) SetError(text string) {
	s.setInfo(text, widget.DangerImportance)
}

func (s *statusBarArea) ClearInfo() {
	s.SetInfo("")
}

func (s *statusBarArea) setInfo(text string, importance widget.Importance) {
	s.infoText.Text = text
	s.infoText.Importance = importance
	s.infoText.Refresh()
}
