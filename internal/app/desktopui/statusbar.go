package desktopui

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
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
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

type DesktopUI interface {
	app.UI

	ShowManageCharactersWindow()
}

type StatusBar struct {
	widget.BaseWidget

	characterCount *StatusBarItem
	eveClock       *StatusBarItem
	eveStatus      *StatusBarItem
	eveStatusError string
	infoText       *widget.Label
	newVersionHint *fyne.Container
	u              DesktopUI
	updateStatus   *StatusBarItem
}

func NewStatusBar(u DesktopUI) *StatusBar {
	a := &StatusBar{
		infoText:       widget.NewLabel(""),
		newVersionHint: container.NewHBox(),
		u:              u,
	}
	a.ExtendBaseWidget(a)
	a.characterCount = NewStatusBarItem(theme.NewThemedResource(icons.GroupSvg), "?", func() {
		u.ShowManageCharactersWindow()
	})
	a.updateStatus = NewStatusBarItem(theme.NewThemedResource(icons.UpdateSvg), "?", func() {
		u.ShowUpdateStatusWindow()
	})
	a.eveClock = NewStatusBarItem(
		theme.NewThemedResource(icons.AccesstimefilledSvg),
		"?",
		a.showClockDialog,
	)
	a.eveStatus = NewStatusBarItem(theme.MediaRecordIcon(), "?", a.showEveStatusDialog)
	return a
}

func (a *StatusBar) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox(
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
	return widget.NewSimpleRenderer(c)
}

func (a *StatusBar) showClockDialog() {
	content := widget.NewRichTextFromMarkdown("")
	d := dialog.NewCustom("EVE Clock", "Close", content, a.u.MainWindow())
	a.u.ModifyShortcutsForDialog(d, a.u.MainWindow())
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

func (a *StatusBar) showEveStatusDialog() {
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
	d := dialog.NewCustom("ESI status", "OK", lb, a.u.MainWindow())
	a.u.ModifyShortcutsForDialog(d, a.u.MainWindow())
	d.Show()
	d.Resize(fyne.Size{Width: 400, Height: 200})
}

func (a *StatusBar) StartUpdateTicker() {
	clockTicker := time.NewTicker(clockUpdateTicker)
	go func() {
		for {
			t := time.Now().UTC().Format("15:04")
			a.eveClock.SetText(t)
			<-clockTicker.C
		}
	}()
	if a.u.IsOffline() {
		a.setEveStatus(eveStatusOffline, "OFFLINE", "Offline mode")
		a.updateUpdateStatus()
		return
	}
	updateTicker := time.NewTicker(characterUpdateStatusTicker)
	go func() {
		for {
			a.updateUpdateStatus()
			<-updateTicker.C
		}
	}()
	esiStatusTicker := time.NewTicker(esiStatusUpdateTicker)
	go func() {
		for {
			x, err := a.u.ESIStatusService().Fetch(context.TODO())
			var t, errorMessage string
			var s eveStatus
			if err != nil {
				slog.Error("Failed to fetch ESI status", "err", err)
				errorMessage = a.u.ErrorDisplay(err)
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
		l := iwidget.NewCustomHyperlink("Update available", func() {
			c := container.NewVBox(
				container.NewHBox(widget.NewLabel("Latest version:"), layout.NewSpacer(), widget.NewLabel(v.Latest)),
				container.NewHBox(widget.NewLabel("You have:"), layout.NewSpacer(), widget.NewLabel(v.Local)),
			)
			u := a.u.WebsiteRootURL().JoinPath("releases")
			d := dialog.NewCustomConfirm("Update available", "Download", "Close", c, func(ok bool) {
				if !ok {
					return
				}
				if err := a.u.App().OpenURL(u); err != nil {
					a.u.ShowErrorDialog("Failed to open download page", err, a.u.MainWindow())
				}
			}, a.u.MainWindow(),
			)
			a.u.ModifyShortcutsForDialog(d, a.u.MainWindow())
			d.Show()
		})
		a.newVersionHint.Add(widget.NewSeparator())
		a.newVersionHint.Add(l)
	}()
}

func (a *StatusBar) Update() {
	x := a.u.StatusCacheService().ListCharacters()
	a.characterCount.SetText(strconv.Itoa(len(x)))
	a.updateUpdateStatus()
}

func (a *StatusBar) updateUpdateStatus() {
	x := a.u.StatusCacheService().Summary()
	a.updateStatus.SetTextAndImportance(x.Display(), x.Status().ToImportance())
}

func (a *StatusBar) setEveStatus(status eveStatus, title, errorMessage string) {
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

func (s *StatusBar) SetInfo(text string) {
	s.setInfo(text, widget.MediumImportance)
}

func (s *StatusBar) SetError(text string) {
	s.setInfo(text, widget.DangerImportance)
}

func (s *StatusBar) ClearInfo() {
	s.SetInfo("")
}

func (s *StatusBar) setInfo(text string, importance widget.Importance) {
	s.infoText.Text = text
	s.infoText.Importance = importance
	s.infoText.Refresh()
}
