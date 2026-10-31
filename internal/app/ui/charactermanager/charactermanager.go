// Package charactermanager provides a window for managing Characters.
package charactermanager

import (
	"context"

	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
)

type baseUI interface {
	Character() *characterservice.CharacterService
	Corporation() *corporationservice.CorporationService
	CurrentCharacter() *app.Character
	ErrorDisplay(err error) string
	EVEImage() ui.EVEImageService
	GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool)
	GetOrCreateWindowWithOnClosed(id string, titles ...string) (window fyne.Window, created bool, onClosed func())
	Go(job func(ctx context.Context)) bool
	IsDeveloperMode() bool
	IsMobile() bool
	IsOffline() bool
	IsUpdateDisabled() bool
	LoadCharacter(ctx context.Context, id int64) error
	MainWindow() fyne.Window
	SetAnyCharacter(ctx context.Context) error
	Signals() *app.Signals
}

// runAsync runs f in a new goroutine.
//
// Tests replace it to run f synchronously.
var runAsync = func(f func()) { go f() }

// runSequenced runs f after all jobs previously queued on g.
//
// Tests replace it to run f synchronously.
var runSequenced = func(g *xsync.SequenceGroup, f func()) { g.Go(f) }

func Show(u baseUI) {
	w, created, onClosed := u.GetOrCreateWindowWithOnClosed("characterWindow", "Manage Characters")
	if !created {
		w.Show()
		return
	}
	cw := newManageCharacters(u, w)
	w.SetContent(fynetooltip.AddWindowToolTipLayer(cw, w.Canvas()))
	w.Resize(fyne.Size{Width: 700, Height: 500})
	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		cw.stop()
	})
	w.SetCloseIntercept(func() {
		w.Close()
		fynetooltip.DestroyWindowToolTipLayer(w.Canvas())
	})
	w.Show()
	cw.update(context.Background())
}

type manageCharacters struct {
	widget.BaseWidget

	characterAdmin    *admin
	characterTags     *manageTags
	characterTraining *training
	displaySnackbar   func(string)
	sb                *kxwidget.Snackbar
	u                 baseUI
	w                 fyne.Window
}

func newManageCharacters(u baseUI, w fyne.Window) *manageCharacters {
	a := &manageCharacters{
		sb: kxwidget.NewSnackbar(w.Canvas()),
		u:  u,
		w:  w,
	}
	a.ExtendBaseWidget(a)
	a.displaySnackbar = a.sb.Display
	a.characterAdmin = newAdmin(a)
	a.characterTags = newManageTags(a)
	a.characterTraining = newTraining(a)
	a.sb.Start()
	return a
}

func (a *manageCharacters) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewAppTabs(
		ui.NewTabItem("Characters", a.characterAdmin),
		ui.NewTabItem("Tags", a.characterTags),
		ui.NewTabItem("Training", a.characterTraining),
	)
	c.SetTabLocation(container.TabLocationLeading)
	return widget.NewSimpleRenderer(c)
}

func (a *manageCharacters) stop() {
	a.sb.Stop()
	a.characterTags.stop()
	a.characterTraining.stop()
}

func (a *manageCharacters) update(ctx context.Context) {
	runAsync(func() {
		a.characterAdmin.update(ctx)
	})
	runAsync(func() {
		a.characterTags.update(ctx)
	})
	runAsync(func() {
		a.characterTraining.update(ctx)
	})
}

func (a *manageCharacters) reportError(text string, err error) {
	slog.Error(text, "error", err)
	a.displaySnackbar(fmt.Sprintf("ERROR: %s: %s", text, err))
}
