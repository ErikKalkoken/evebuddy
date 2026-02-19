package ui

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/github"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	clockUpdateTicker = 2 * time.Second
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

	characterCount    *statusBarItem
	eveClock          *statusBarItem
	eveStatus         *statusBarItem
	eveStatusError    string
	u                 *DesktopUI
	updateHint        *updateHint
	updateStatus      *statusBarItem
	updatingIndicator *iwidget.Activity
}

func newStatusBar(u *DesktopUI) *statusBar {
	ac := iwidget.NewActivity()
	ac.SetToolTip("Synchronizing with game server...")
	a := &statusBar{
		updatingIndicator: ac,
		u:                 u,
	}
	a.ExtendBaseWidget(a)
	warningIcon := ttwidget.NewIcon(theme.NewWarningThemedResource(theme.WarningIcon()))
	warningIcon.Hide()
	a.characterCount = newStatusBarItemWithTrailing(
		theme.NewThemedResource(icons.GroupSvg),
		warningIcon,
		"?",
		func() {
			showManageCharactersWindow(u.baseUI)
		},
	)
	a.characterCount.SetToolTip("Number of characters - click to manage")
	a.u.onUpdateMissingScope = func(characterCount int) {
		fyne.Do(func() {
			if characterCount > 0 {
				warningIcon.SetToolTip(fmt.Sprintf("%d character(s) missing scope", characterCount))
				warningIcon.Show()
			} else {
				warningIcon.Hide()
			}
		})
	}

	spacer := newSpacer(a.updatingIndicator.MinSize())
	a.updateStatus = newStatusBarItemWithTrailing(
		theme.NewThemedResource(icons.UpdateSvg),
		container.NewStack(spacer, a.updatingIndicator),
		"?",
		func() {
			showUpdateStatusWindow(u.baseUI)
		},
	)
	a.updateStatus.SetToolTip("Current update status - click for details")
	a.eveClock = newStatusBarItem(
		theme.NewThemedResource(icons.AccesstimefilledSvg),
		"?",
		a.showClockDialog,
	)
	a.eveClock.SetToolTip("Current EVE time - click to enlarge")
	a.eveStatus = newStatusBarItem(theme.MediaRecordIcon(), "?", a.showEveStatusDialog)
	a.eveStatus.SetToolTip("EVE server status - click for details")
	a.updateHint = newUpdateHint(u)
	a.updateHint.Hide()
	return a
}

func (a *statusBar) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox(
			a.u.statusText,
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

func (a *statusBar) start() {
	// signals
	a.u.characterAdded.AddListener(func(_ context.Context, _ *app.Character) {
		a.updateCharacterCount()
		a.updateUpdateStatus()
	})
	a.u.characterRemoved.AddListener(func(_ context.Context, _ *app.EntityShort) {
		a.updateCharacterCount()
		a.updateUpdateStatus()
	})
	a.u.characterSectionUpdated.AddListener(func(_ context.Context, _ characterSectionUpdated) {
		a.updateUpdateStatus()
	})
	a.u.corporationSectionUpdated.AddListener(func(_ context.Context, _ corporationSectionUpdated) {
		a.updateUpdateStatus()
	})
	a.u.generalSectionUpdated.AddListener(func(_ context.Context, _ generalSectionUpdated) {
		a.updateUpdateStatus()
	})

	var mu sync.Mutex
	var updating set.Set[string]
	showUpdate := func(on bool) {
		if on {
			a.updatingIndicator.Start()
			a.updatingIndicator.Show()
		} else {
			a.updatingIndicator.Hide()
			a.updatingIndicator.Stop()
		}
		a.updateStatus.Refresh()
	}
	a.u.updateStarted.AddListener(func(_ context.Context, id string) {
		var on bool
		mu.Lock()
		updating.Add(id)
		on = updating.Size() > 0
		mu.Unlock()
		fyne.Do(func() {
			showUpdate(on)
		})
	})
	a.u.updateStopped.AddListener(func(_ context.Context, id string) {
		var on bool
		mu.Lock()
		updating.Delete(id)
		on = updating.Size() > 0
		mu.Unlock()
		fyne.Do(func() {
			showUpdate(on)
		})
	})
	fyne.Do(func() {
		a.updateCharacterCount()
		a.updateUpdateStatus()
		a.refreshEveStatus()
	})

	clockTicker := time.NewTicker(clockUpdateTicker)
	go func() {
		for {
			fyne.Do(func() {
				a.eveClock.SetText(time.Now().UTC().Format("15:04"))
			})
			<-clockTicker.C
		}
	}()

	if a.u.IsOffline() {
		fyne.Do(func() {
			a.setEveStatus(eveStatusOffline, "OFFLINE", "Offline mode")
		})
		return
	}

	a.u.refreshTickerExpired.AddListener(func(ctx context.Context, s struct{}) {
		a.refreshEveStatus()
	})

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

func (a *statusBar) refreshEveStatus() {
	var t, errorMessage string
	var s eveStatus
	if a.u.ess.IsDailyDowntime() {
		s = eveStatusOffline
		t = "OFFLINE"
		errorMessage = fmt.Sprintf(
			"Offline during planned daily downtime:\n%s",
			a.u.ess.DailyDowntime(),
		)
	} else {
		x, err := a.u.ess.Fetch(context.Background())
		if err != nil {
			slog.Error("Failed to fetch ESI status", "err", err)
			errorMessage = a.u.humanizeError(err)
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
	}
	fyne.Do(func() {
		a.setEveStatus(s, t, errorMessage)
	})
}

func (a *statusBar) updateCharacterCount() {
	s := strconv.Itoa(len(a.u.scs.ListCharacters()))
	fyne.Do(func() {
		a.characterCount.SetText(s)
	})
}

func (a *statusBar) updateUpdateStatus() {
	var s string
	var i widget.Importance
	if a.u.isUpdateDisabled.Load() || a.u.ess.IsDailyDowntime() {
		s = "Off"
	} else {
		x := a.u.scs.Summary()
		s = x.DisplayShort()
		i = x.Status().ToImportance()
	}
	fyne.Do(func() {
		a.updateStatus.SetTextAndImportance(s, i)
	})
}

func (a *statusBar) showClockDialog() {
	clock := widget.NewLabel("")
	clock.SizeName = theme.SizeNameHeadingText
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
	a.eveStatus.SetLeading(r2)
	a.eveStatus.SetText(title)
}

// statusBarItem is a widget with a label and an optional icon, which can be tapped.
type statusBarItem struct {
	ttwidget.ToolTipWidget

	// The function that is called when the label is tapped.
	OnTapped func()

	bg       *canvas.Rectangle
	label    *widget.Label
	leading  *widget.Icon
	trailing fyne.CanvasObject
}

var _ fyne.Tappable = (*statusBarItem)(nil)
var _ desktop.Hoverable = (*statusBarItem)(nil)

func newStatusBarItem(leading fyne.Resource, text string, tapped func()) *statusBarItem {
	return newStatusBarItemWithTrailing(leading, nil, text, tapped)
}

func newStatusBarItemWithTrailing(leading fyne.Resource, trailing fyne.CanvasObject, text string, tapped func()) *statusBarItem {
	icon := widget.NewIcon(icons.BlankSvg)
	if leading != nil {
		icon.SetResource(leading)
	} else {
		icon.Hide()
	}
	if trailing == nil {
		trailing = canvas.NewRectangle(color.Transparent)
		trailing.Hide()
	}
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameHover))
	bg.Hide()
	w := &statusBarItem{
		bg:       bg,
		label:    widget.NewLabel(text),
		leading:  icon,
		OnTapped: tapped,
		trailing: trailing,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *statusBarItem) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.NewStack(
		w.bg,
		container.New(layout.NewCustomPaddedLayout(0, 0, p, p),
			container.New(layout.NewCustomPaddedHBoxLayout(0),
				container.New(layout.NewCustomPaddedLayout(0, 0, p, 0), w.leading),
				w.label,
				w.trailing,
			)),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *statusBarItem) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(theme.ColorNameHover, v)
	w.bg.Refresh()
	w.leading.Refresh()
	w.label.Refresh()
	w.BaseWidget.Refresh()
}

// SetLeading updates the leading icon.
func (w *statusBarItem) SetLeading(icon fyne.Resource) {
	w.leading.SetResource(icon)
}

// SetText updates the label's text.
func (w *statusBarItem) SetText(text string) {
	w.SetTextAndImportance(text, widget.MediumImportance)
}

// SetText updates the label's text and importance.
func (w *statusBarItem) SetTextAndImportance(text string, importance widget.Importance) {
	w.label.Text = text
	w.label.Importance = importance
	w.label.Refresh()
}

func (w *statusBarItem) Tapped(_ *fyne.PointEvent) {
	if w.OnTapped != nil {
		w.OnTapped()
	}
}

func (w *statusBarItem) TappedSecondary(_ *fyne.PointEvent) {
}

func (w *statusBarItem) MouseIn(e *desktop.MouseEvent) {
	w.ToolTipWidget.MouseIn(e)
	if w.OnTapped != nil {
		w.bg.Show()
	}
}

func (w *statusBarItem) MouseMoved(e *desktop.MouseEvent) {
	w.ToolTipWidget.MouseMoved(e)
}

func (w *statusBarItem) MouseOut() {
	w.ToolTipWidget.MouseOut()
	w.bg.Hide()
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
				w.u.showErrorDialog("Failed to open download page", err, w.u.MainWindow())
			}
		}, w.u.MainWindow(),
		)
		w.u.ModifyShortcutsForDialog(d, w.u.MainWindow())
		d.Show()
	})
	c := container.NewHBox(l, widget.NewSeparator())
	return widget.NewSimpleRenderer(c)
}
