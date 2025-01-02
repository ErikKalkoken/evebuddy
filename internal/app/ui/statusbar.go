package ui

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/ErikKalkoken/evebuddy/internal/github"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
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
	characterCount *widgets.StatusBarItem
	content        *fyne.Container
	eveClock       *widgets.StatusBarItem
	eveStatus      *widgets.StatusBarItem
	eveStatusError string
	infoText       *widget.Label
	newVersionHint *fyne.Container
	u              *UI
	updateStatus   *widgets.StatusBarItem
}

func (u *UI) newStatusBarArea() *statusBarArea {
	a := &statusBarArea{
		infoText:       widget.NewLabel(""),
		newVersionHint: container.NewHBox(),
		u:              u,
	}
	a.characterCount = widgets.NewStatusBarItem(theme.AccountIcon(), "?", func() {
		u.showAccountDialog()
	})
	a.updateStatus = widgets.NewStatusBarItem(theme.NewThemedResource(resourceUpdateSvg), "?", func() {
		u.showStatusWindow()
	})
	a.eveClock = widgets.NewStatusBarItem(
		theme.NewThemedResource(resourceAccesstimefilledSvg),
		"?",
		a.showClockDialog,
	)
	a.eveStatus = widgets.NewStatusBarItem(theme.MediaRecordIcon(), "?", a.showDetail)
	a.content = container.NewVBox(widget.NewSeparator(), container.NewHBox(
		a.infoText,
		layout.NewSpacer(),
		a.newVersionHint,
		widget.NewSeparator(),
		a.updateStatus,
		widget.NewSeparator(),
		a.characterCount,
		widget.NewSeparator(),
		a.eveClock,
		widget.NewSeparator(),
		a.eveStatus,
	))
	return a
}

func (a *statusBarArea) showClockDialog() {
	content := widget.NewRichTextFromMarkdown("")
	d := dialog.NewCustom("EVE Clock", "Close", content, a.u.window)
	stop := make(chan struct{})
	timer := time.NewTicker(1 * time.Second)
	go func() {
		for {
			s := time.Now().UTC().Format("15:04:05")
			content.ParseMarkdown(fmt.Sprintf("# %s", s))
			select {
			case <-stop:
				return
			case <-timer.C:
			}
		}
	}()
	d.SetOnClosed(func() {
		stop <- struct{}{}
	})
	d.Show()
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
	d := dialog.NewCustom("ESI status", "OK", lb, a.u.window)
	d.Show()
	d.Resize(fyne.Size{Width: 400, Height: 200})
}

func (a *statusBarArea) StartUpdateTicker() {
	clockTicker := time.NewTicker(clockUpdateTicker)
	go func() {
		for {
			t := time.Now().UTC().Format("15:04")
			a.eveClock.SetText(t)
			<-clockTicker.C
		}
	}()
	if a.u.IsOffline {
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
			x, err := a.u.ESIStatusService.Fetch(context.TODO())
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
		current := a.u.fyneApp.Metadata().Version
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
		a.newVersionHint.Add(widget.NewSeparator())
		a.newVersionHint.Add(l)
	}()
}

func (a *statusBarArea) refreshCharacterCount() {
	x := a.u.StatusCacheService.ListCharacters()
	a.characterCount.SetText(strconv.Itoa(len(x)))
}

func (a *statusBarArea) refreshUpdateStatus() {
	x := a.u.StatusCacheService.Summary()
	a.updateStatus.SetTextAndImportance(x.Display(), x.Status().ToImportance())
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
