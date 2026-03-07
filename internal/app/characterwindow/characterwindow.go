// Package characterwindow provides a window for managing Characters.
package characterwindow

import (
	"context"

	"fmt"
	"log/slog"

	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"fyne.io/fyne/v2/widget"

	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type uiServices interface {
	Character() *characterservice.CharacterService
	Corporation() *corporationservice.CorporationService
	CurrentCharacter() *app.Character
	CurrentCorporation() *app.Corporation
	ErrorDisplay(err error) string
	EVEImage() *eveimageservice.EVEImageService
	GetOrCreateWindowWithOnClosed(id string, titles ...string) (window fyne.Window, created bool, onClosed func())
	IsMobile() bool
	IsDeveloperMode() bool
	IsOfflineMode() bool
	IsUpdateDisabled() bool
	LoadCharacter(ctx context.Context, id int64) error
	LoadCorporation(ctx context.Context, id int64) error
	SetAnyCharacter(ctx context.Context) error
	SetAnyCorporation(ctx context.Context) error
	Signals() *app.Signals
}

func Show(u uiServices) {
	w, created, onClosed := u.GetOrCreateWindowWithOnClosed("characterWindow", "Manage Characters")
	if !created {
		w.Show()
		return
	}
	cw := newCharacterWindow(u, w)
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
	go cw.update(context.Background())
}

type characterWindow struct {
	widget.BaseWidget

	characterAdmin    *characterAdmin
	characterTags     *characterTags
	characterTraining *characterTraining
	u                 uiServices
	sb                *xwidget.Snackbar
	w                 fyne.Window
}

func newCharacterWindow(u uiServices, w fyne.Window) *characterWindow {
	a := &characterWindow{
		sb: xwidget.NewSnackbar(w),
		u:  u,
		w:  w,
	}
	a.ExtendBaseWidget(a)
	a.characterAdmin = newCharacterAdmin(a)
	a.characterTags = newCharacterTags(a)
	a.characterTraining = newCharacterTraining(a)
	a.sb.Start()
	return a
}

func (a *characterWindow) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewAppTabs(
		container.NewTabItem("Characters", a.characterAdmin),
		container.NewTabItem("Tags", a.characterTags),
		container.NewTabItem("Training", a.characterTraining),
	)
	c.SetTabLocation(container.TabLocationLeading)
	return widget.NewSimpleRenderer(c)
}

func (a *characterWindow) stop() {
	a.sb.Stop()
}

func (a *characterWindow) update(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Go(func() {
		a.characterAdmin.update(ctx)
	})
	wg.Go(func() {
		a.characterTags.update(ctx)
	})
	wg.Go(func() {
		a.characterTraining.update(ctx)
	})
	wg.Wait()
}

func (a *characterWindow) reportError(text string, err error) {
	slog.Error(text, "error", err)
	a.sb.Show(fmt.Sprintf("ERROR: %s: %s", text, err))
}
