package ui

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/github"
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

type statusBar struct {
	widget.BaseWidget

	characterCount *StatusBarItem
	eveClock       *StatusBarItem
	eveStatus      *StatusBarItem
	eveStatusError string
	infoText       *widget.Label
	updateHint     *updateHint
	u              *DesktopUI
	updateStatus   *StatusBarItem
	latestVersion  *widget.Label
	currentVersion *widget.Label
}

func newStatusBar(u *DesktopUI) *statusBar {
	a := &statusBar{
		infoText: widget.NewLabel(""),
		u:        u,
	}
	a.ExtendBaseWidget(a)
	a.characterCount = NewStatusBarItem(theme.NewThemedResource(icons.GroupSvg), "?", func() {
		u.showManageCharactersWindow()
	})
	a.updateStatus = NewStatusBarItem(theme.NewThemedResource(icons.UpdateSvg), "?", func() {
		u.showUpdateStatusWindow()
	})
	a.eveClock = NewStatusBarItem(
		theme.NewThemedResource(icons.AccesstimefilledSvg),
		"?",
		a.showClockDialog,
	)
	a.eveStatus = NewStatusBarItem(theme.MediaRecordIcon(), "?", a.showEveStatusDialog)
	a.updateHint = newUpdateHint(u)
	a.updateHint.Hide()
	return a
}

func (a *statusBar) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox(
			a.infoText,
			layout.NewSpacer(),
			a.updateHint,
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

func (a *statusBar) showClockDialog() {
	clock := iwidget.NewLabelWithSize("", theme.SizeNameHeadingText)
	d := dialog.NewCustom("EVE Clock", "Close", clock, a.u.MainWindow())
	a.u.ModifyShortcutsForDialog(d, a.u.MainWindow())
	stop := make(chan struct{})
	timer := time.NewTicker(1 * time.Second)
	go func() {
		for {
			s := time.Now().UTC().Format("15:04:05")
			fyne.Do(func() {
				clock.SetText(s)
			})
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

func (a *statusBar) showEveStatusDialog() {
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

func (a *statusBar) startUpdateTicker() {
	clockTicker := time.NewTicker(clockUpdateTicker)
	go func() {
		for {
			t := time.Now().UTC().Format("15:04")
			fyne.Do(func() {
				a.eveClock.SetText(t)
			})
			<-clockTicker.C
		}
	}()
	if a.u.IsOffline() {
		fyne.Do(func() {
			a.setEveStatus(eveStatusOffline, "OFFLINE", "Offline mode")
			a.refreshUpdateStatus()
		})
		return
	}
	updateTicker := time.NewTicker(characterUpdateStatusTicker)
	go func() {
		for {
			fyne.Do(func() {
				a.refreshUpdateStatus()
			})
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
			fyne.Do(func() {
				a.setEveStatus(s, t, errorMessage)
			})
			<-esiStatusTicker.C
		}
	}()
	if !a.u.IsOffline() {
		go func() {
			v, err := a.u.availableUpdate()
			if err != nil {
				slog.Error("fetch latest github version for download hint", "err", err)
				return
			}
			if !v.IsRemoteNewer {
				return
			}
			fyne.Do(func() {
				a.updateHint.set(v)
				a.updateHint.Show()
			})
		}()
	}
}

func (a *statusBar) update() {
	x := a.u.StatusCacheService().ListCharacters()
	fyne.Do(func() {
		a.characterCount.SetText(strconv.Itoa(len(x)))
		a.refreshUpdateStatus()
	})
}

func (a *statusBar) refreshUpdateStatus() {
	x := a.u.StatusCacheService().Summary()
	a.updateStatus.SetTextAndImportance(x.Display(), x.Status().ToImportance())
}

func (a *statusBar) setEveStatus(status eveStatus, title, errorMessage string) {
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

func (s *statusBar) SetInfo(text string) {
	s.setInfo(text, widget.MediumImportance)
}

func (s *statusBar) setInfo(text string, importance widget.Importance) {
	s.infoText.Text = text
	s.infoText.Importance = importance
	s.infoText.Refresh()
}

// StatusBarItem is a widget with a label and an optional icon, which can be tapped.
type StatusBarItem struct {
	widget.BaseWidget
	icon  *widget.Icon
	label *widget.Label

	// The function that is called when the label is tapped.
	OnTapped func()

	hovered bool
}

var _ fyne.Tappable = (*StatusBarItem)(nil)
var _ desktop.Hoverable = (*StatusBarItem)(nil)

func NewStatusBarItem(res fyne.Resource, text string, tapped func()) *StatusBarItem {
	w := &StatusBarItem{OnTapped: tapped, label: widget.NewLabel(text)}
	if res != nil {
		w.icon = widget.NewIcon(res)
	}
	w.ExtendBaseWidget(w)
	return w
}

// SetResource updates the icon's resource
func (w *StatusBarItem) SetResource(icon fyne.Resource) {
	w.icon.SetResource(icon)
}

// SetText updates the label's text
func (w *StatusBarItem) SetText(text string) {
	w.label.SetText(text)
}

// SetText updates the label's text and importance
func (w *StatusBarItem) SetTextAndImportance(text string, importance widget.Importance) {
	w.label.Text = text
	w.label.Importance = importance
	w.label.Refresh()
}

func (w *StatusBarItem) Tapped(_ *fyne.PointEvent) {
	if w.OnTapped != nil {
		w.OnTapped()
	}
}

func (w *StatusBarItem) TappedSecondary(_ *fyne.PointEvent) {
}

// Cursor returns the cursor type of this widget
func (w *StatusBarItem) Cursor() desktop.Cursor {
	if w.hovered {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// MouseIn is a hook that is called if the mouse pointer enters the element.
func (w *StatusBarItem) MouseIn(e *desktop.MouseEvent) {
	w.hovered = true
}

func (w *StatusBarItem) MouseMoved(*desktop.MouseEvent) {
	// needed to satisfy the interface only
}

// MouseOut is a hook that is called if the mouse pointer leaves the element.
func (w *StatusBarItem) MouseOut() {
	w.hovered = false
}

func (w *StatusBarItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewHBox()
	if w.icon != nil {
		c.Add(w.icon)
	}
	c.Add(w.label)
	return widget.NewSimpleRenderer(c)
}

type updateHint struct {
	widget.BaseWidget

	latest  *widget.Label
	current *widget.Label
	u       *DesktopUI
}

func newUpdateHint(u *DesktopUI) *updateHint {
	w := &updateHint{
		latest:  widget.NewLabel(""),
		current: widget.NewLabel(""),
		u:       u,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *updateHint) set(v github.VersionInfo) {
	w.current.SetText(v.Local)
	w.latest.SetText(v.Latest)
}

func (w *updateHint) CreateRenderer() fyne.WidgetRenderer {
	l := iwidget.NewCustomHyperlink("Update available", func() {
		c := container.NewVBox(
			container.NewHBox(widget.NewLabel("Latest version:"), layout.NewSpacer(), w.latest),
			container.NewHBox(widget.NewLabel("You have:"), layout.NewSpacer(), w.current),
		)
		u := w.u.websiteRootURL().JoinPath("releases")
		d := dialog.NewCustomConfirm("Update available", "Download", "Close", c, func(ok bool) {
			if !ok {
				return
			}
			if err := w.u.App().OpenURL(u); err != nil {
				w.u.ShowErrorDialog("Failed to open download page", err, w.u.MainWindow())
			}
		}, w.u.MainWindow(),
		)
		w.u.ModifyShortcutsForDialog(d, w.u.MainWindow())
		d.Show()
	})
	c := container.NewHBox(l, widget.NewSeparator())
	return widget.NewSimpleRenderer(c)
}
