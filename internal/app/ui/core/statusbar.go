package core

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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/managecharacters"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/updatestatus"
	"github.com/ErikKalkoken/evebuddy/internal/github"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	clockUpdateTicker = 2 * time.Second
	versionTicker     = 3600 * time.Second
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
	updatingIndicator *xwidget.Activity
}

func newStatusBar(u *DesktopUI) *statusBar {
	ac := xwidget.NewActivity()
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
			managecharacters.Show(u)
		},
	)
	a.characterCount.SetToolTip("Number of characters - click to manage")
	a.u.onUpdateMissingScope = func(characterCount int) {
		if characterCount > 0 {
			warningIcon.SetToolTip(fmt.Sprintf("%d character(s) missing scope", characterCount))
			warningIcon.Show()
		} else {
			warningIcon.Hide()
		}
	}

	spacer := xwidget.NewSpacer(a.updatingIndicator.MinSize())
	a.updateStatus = newStatusBarItemWithTrailing(
		theme.NewThemedResource(icons.UpdateSvg),
		container.NewStack(spacer, a.updatingIndicator),
		"?",
		func() {
			updatestatus.Show(u)
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
	a.updateHint = newUpdateHint(u.IsDeveloperMode(), u.MainWindow())
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
	a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.updateCharacterCount(ctx)
		a.updateUpdateStatus(ctx)
	})
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.updateCharacterCount(ctx)
		a.updateUpdateStatus(ctx)
	})
	a.u.Signals().CharacterSectionUpdated.AddListener(func(ctx context.Context, _ app.CharacterSectionUpdated) {
		a.updateUpdateStatus(ctx)
	})
	a.u.Signals().CorporationSectionUpdated.AddListener(func(ctx context.Context, _ app.CorporationSectionUpdated) {
		a.updateUpdateStatus(ctx)
	})
	a.u.Signals().EveUniverseSectionUpdated.AddListener(func(ctx context.Context, _ app.EveUniverseSectionUpdated) {
		a.updateUpdateStatus(ctx)
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
	a.u.Signals().UpdateStarted.AddListener(func(_ context.Context, id string) {
		var on bool
		mu.Lock()
		updating.Add(id)
		on = updating.Size() > 0
		mu.Unlock()
		fyne.Do(func() {
			showUpdate(on)
		})
	})
	a.u.Signals().UpdateStopped.AddListener(func(_ context.Context, id string) {
		var on bool
		mu.Lock()
		updating.Delete(id)
		on = updating.Size() > 0
		mu.Unlock()
		fyne.Do(func() {
			showUpdate(on)
		})
	})
	ctx := context.Background()
	a.updateCharacterCount(ctx)
	a.updateUpdateStatus(ctx)
	a.updateEveStatus(ctx)

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

	a.u.Signals().RefreshTickerExpired.AddListener(func(ctx context.Context, _ struct{}) {
		a.updateEveStatus(ctx)
	})

	tickerNewVersion := time.NewTicker(versionTicker)
	go func() {
		for {
			func() {
				v, err := a.u.availableUpdate(ctx)
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
			<-tickerNewVersion.C
		}
	}()
}

func (a *statusBar) updateEveStatus(ctx context.Context) {
	set := func(status eveStatus, title string, errorMessage string) {
		fyne.Do(func() {
			a.setEveStatus(status, title, errorMessage)
		})
	}

	if a.u.ess.IsDailyDowntime() {
		s := fmt.Sprintf(
			"Offline during planned daily downtime:\n%s",
			a.u.ess.DailyDowntime(),
		)
		set(eveStatusOffline, "OFFLINE", s)
		a.u.isOffline.Store(true)
		return
	}

	status, err := a.u.ess.Fetch(ctx)
	if err != nil {
		slog.Error("Failed to fetch ESI status", "err", err)
		set(eveStatusError, "ERROR", a.u.ErrorDisplay(err))
		return
	}
	if !status.IsOK() {
		set(eveStatusOffline, "OFFLINE", status.ErrorMessage)
		a.u.isOffline.Store(true)
		return
	}

	p := message.NewPrinter(language.English)
	set(eveStatusOnline, p.Sprintf("%d players", status.PlayerCount), "")
	a.u.isOffline.Store(false)
}

func (a *statusBar) updateCharacterCount(_ context.Context) {
	s := strconv.Itoa(len(a.u.StatusCache().ListCharacters()))
	fyne.Do(func() {
		a.characterCount.SetText(s)
	})
}

func (a *statusBar) updateUpdateStatus(_ context.Context) {
	if a.u.isUpdateDisabled.Load() || a.u.ess.IsDailyDowntime() {
		fyne.Do(func() {
			a.updateStatus.SetTextAndImportance("OFF", widget.MediumImportance)
		})
		return
	}
	x := a.u.StatusCache().Summary()
	fyne.Do(func() {
		a.updateStatus.SetTextAndImportance(x.DisplayShort(), x.Status().ToImportance())
	})
}

func (a *statusBar) showClockDialog() {
	clock := widget.NewLabel("")
	clock.SizeName = theme.SizeNameHeadingText
	d := dialog.NewCustom("EVE Clock", "Close", clock, a.u.MainWindow())
	xdesktop.DisableShortcutsForDialog(d, a.u.MainWindow())

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
	xdesktop.DisableShortcutsForDialog(d, a.u.MainWindow())
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

	current         *widget.Label
	isDeveloperMode bool
	latest          *widget.Label
	window          fyne.Window
}

func newUpdateHint(isDeveloperMode bool, window fyne.Window) *updateHint {
	w := &updateHint{
		current:         widget.NewLabel(""),
		isDeveloperMode: isDeveloperMode,
		latest:          widget.NewLabel(""),
		window:          window,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *updateHint) set(v github.VersionInfo) {
	w.current.SetText(v.Local)
	w.latest.SetText(v.Latest)
}

func (w *updateHint) CreateRenderer() fyne.WidgetRenderer {
	l := xwidget.NewCustomHyperlink("Update available", func() {
		c := container.NewVBox(
			container.NewHBox(widget.NewLabel("Latest version:"), layout.NewSpacer(), w.latest),
			container.NewHBox(widget.NewLabel("You have:"), layout.NewSpacer(), w.current),
			xwidget.NewStandardSpacer(),
		)
		u := ui.WebsiteRootURL().JoinPath("releases")
		d := dialog.NewCustomConfirm(
			"Update available", "Download", "Close", c, func(ok bool) {
				if !ok {
					return
				}
				if err := fyne.CurrentApp().OpenURL(u); err != nil {
					ui.ShowErrorAndLog("Failed to open download page", err, w.isDeveloperMode, w.window)
				}
			},
			w.window,
		)
		xdesktop.DisableShortcutsForDialog(d, w.window)
		d.Show()
	})
	c := container.NewHBox(l, widget.NewSeparator())
	return widget.NewSimpleRenderer(c)
}
