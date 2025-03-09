package desktop

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxdialog "github.com/ErikKalkoken/fyne-kx/dialog"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
)

const (
	characterUpdateStatusTicker = 2 * time.Second
	clockUpdateTicker           = 2 * time.Second
	esiStatusUpdateTicker       = 60 * time.Second
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
	characterCount *appwidget.StatusBarItem
	content        *fyne.Container
	eveClock       *appwidget.StatusBarItem
	eveStatus      *appwidget.StatusBarItem
	eveStatusError string
	infoText       *widget.Label
	newVersionHint *fyne.Container
	u              *DesktopUI
	updateStatus   *appwidget.StatusBarItem
}

func newStatusBarArea(u *DesktopUI) *statusBarArea {
	a := &statusBarArea{
		infoText:       widget.NewLabel(""),
		newVersionHint: container.NewHBox(),
		u:              u,
	}
	a.characterCount = appwidget.NewStatusBarItem(theme.AccountIcon(), "?", func() {
		u.showAccountWindow()
	})
	a.updateStatus = appwidget.NewStatusBarItem(theme.NewThemedResource(icon.UpdateSvg), "?", func() {
		u.ShowUpdateStatusWindow()
	})
	a.eveClock = appwidget.NewStatusBarItem(
		theme.NewThemedResource(icon.AccesstimefilledSvg),
		"?",
		a.showClockDialog,
	)
	a.eveStatus = appwidget.NewStatusBarItem(theme.MediaRecordIcon(), "?", a.showEveStatusDialog)
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
	d := dialog.NewCustom("EVE Clock", "Close", content, a.u.Window)
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

func (a *statusBarArea) showEveStatusDialog() {
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
	d := dialog.NewCustom("ESI status", "OK", lb, a.u.Window)
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
		v, err := a.u.AvailableUpdate()
		if err != nil {
			slog.Error("fetch latest github version for download hint", "err", err)
			return
		}
		if !v.IsRemoteNewer {
			return
		}
		l := ui.NewCustomHyperlink("Update available", func() {
			c := container.NewVBox(
				container.NewHBox(widget.NewLabel("Latest version:"), layout.NewSpacer(), widget.NewLabel(v.Latest)),
				container.NewHBox(widget.NewLabel("You have:"), layout.NewSpacer(), widget.NewLabel(v.Local)),
			)
			u := a.u.WebsiteRootURL().JoinPath("releases")
			d := dialog.NewCustomConfirm("Update available", "Download", "Close", c, func(ok bool) {
				if !ok {
					return
				}
				if err := a.u.FyneApp.OpenURL(u); err != nil {
					d2 := ui.NewErrorDialog("Failed to open download page", err, a.u.Window)
					d2.Show()
				}
			}, a.u.Window,
			)
			kxdialog.AddDialogKeyHandler(d, a.u.Window)
			d.Show()
		})
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
