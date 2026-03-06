// Package characterwindow provides a window for managing Characters.
package characterwindow

import (
	"context"

	"fmt"
	"image/color"
	"log/slog"

	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fynetooltip "github.com/dweymouth/fyne-tooltip"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type EIS interface {
	CharacterPortraitAsync(id int64, size int, setter func(r fyne.Resource))
}

type UIService interface {
	CurrentCharacterID() int64
	CurrentCorporationID() int64
	GetOrCreateWindowWithOnClosed(id string, titles ...string) (window fyne.Window, created bool, onClosed func())
	HasCharacter() bool
	HasCorporation() bool
	HumanizeError(err error) string
	IsOffline() bool
	LoadCharacter(id int64) error
	LoadCorporation(id int64) error
	SetAnyCharacter() error
	SetAnyCorporation() error
}

type Params struct {
	CharacterService   *characterservice.CharacterService
	CorporationService *corporationservice.CorporationService
	EveImageService    EIS
	IsMobile           bool
	IsUpdateDisabled   bool
	Signals            *app.Signals
	UIService          UIService
}

func Show(arg Params) {
	if arg.CharacterService == nil {
		slog.Error("characterWindow: CharacterService missing")
		return
	}
	if arg.CorporationService == nil {
		slog.Error("characterWindow: CorporationService missing")
		return
	}
	if arg.EveImageService == nil {
		slog.Error("characterWindow: EveImageService missing")
		return
	}
	if arg.Signals == nil {
		slog.Error("characterWindow: Signals missing")
		return
	}
	if arg.UIService == nil {
		slog.Error("characterWindow: UIService missing")
		return
	}
	w, created, onClosed := arg.UIService.GetOrCreateWindowWithOnClosed("characterWindow", "Manage Characters")
	if !created {
		w.Show()
		return
	}
	cw := newCharacterWindow(arg, w)
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
	cs                *characterservice.CharacterService
	eis               EIS
	isMobile          bool
	isUpdateDisabled  bool
	rs                *corporationservice.CorporationService
	sb                *xwidget.Snackbar
	signals           *app.Signals
	u                 UIService
	w                 fyne.Window
}

func newCharacterWindow(arg Params, w fyne.Window) *characterWindow {
	a := &characterWindow{
		cs:               arg.CharacterService,
		eis:              arg.EveImageService,
		isMobile:         arg.IsMobile,
		rs:               arg.CorporationService,
		sb:               xwidget.NewSnackbar(w),
		u:                arg.UIService,
		w:                w,
		isUpdateDisabled: arg.IsUpdateDisabled,
		signals:          arg.Signals,
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
func newStandardSpacer() fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSquareSize(theme.Padding()))
	return r
}
