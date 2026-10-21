package charactermanager

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// training is a UI component that allows to configure training watchers for characters.
type training struct {
	widget.BaseWidget

	characters     []*app.Character
	charactersLoad atomic.Int64 // identifies the newest characters load
	cw             *manageCharacters
	list           *widget.List
	signalKey      string
}

func newTraining(cw *manageCharacters) *training {
	a := &training{
		cw:        cw,
		signalKey: cw.u.Signals().UniqueKey(),
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeList()

	// Signals
	a.cw.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	}, a.signalKey)
	a.cw.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	}, a.signalKey)
	return a
}

func (a *training) stop() {
	a.cw.u.Signals().CharacterAdded.RemoveListener(a.signalKey)
	a.cw.u.Signals().CharacterRemoved.RemoveListener(a.signalKey)
}

func (a *training) CreateRenderer() fyne.WidgetRenderer {
	actions := xwidget.NewIconButtonWithMenu(theme.MoreHorizontalIcon(), fyne.NewMenu("",
		fyne.NewMenuItem("Set to currently trained", func() {
			characterIDs := xslices.Map(a.characters, func(x *app.Character) int64 {
				return x.ID
			})
			runAsync(func() {
				a.setWatchedToCurrentlyTrained(context.Background(), characterIDs)
			})
		}),
		fyne.NewMenuItem("Enable all", func() {
			a.setAllWatched(context.Background(), true)
		}),
		fyne.NewMenuItem("Disable all", func() {
			a.setAllWatched(context.Background(), false)
		}),
	))
	ab := xwidget.NewAppBar("Watched Training", a.list, actions)
	ab.HideBackground = !a.cw.u.IsMobile()
	return widget.NewSimpleRenderer(ab)
}

func (a *training) setWatchedToCurrentlyTrained(ctx context.Context, characterIDs []int64) {
	var failed int
	var lastErr error
	for _, characterID := range characterIDs {
		d, err := a.cw.u.Character().TotalTrainingTime(ctx, characterID)
		if err != nil {
			slog.Error("Failed to set watcher for trained character", "characterID", characterID, "error", err)
			failed++
			lastErr = err
			continue
		}
		fyne.Do(func() {
			id := a.indexOf(characterID)
			if id == -1 {
				return // removed meanwhile
			}
			a.updateCharacterWatched(ctx, id, d.ValueOrZero() > 0)
		})
	}
	if failed > 0 {
		a.cw.displaySnackbar(fmt.Sprintf("Failed to set watcher for %s: %s", ihumanize.Plural(failed, "character", "characters"), a.cw.u.ErrorDisplay(lastErr)))
	}
}

func (a *training) setAllWatched(ctx context.Context, on bool) {
	for id := range a.characters {
		a.updateCharacterWatched(ctx, id, on)
	}
}

func (a *training) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			character := ui.NewEveEntityListItem(a.cw.u.EVEImage().EveEntityLogoAsync)
			character.IsAvatar = true
			return container.NewBorder(
				nil,
				nil,
				nil,
				kxwidget.NewSwitch(nil),
				character,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.characters) {
				return
			}
			r := a.characters[id]
			border := co.(*fyne.Container).Objects

			border[0].(*ui.EveEntityListItem).Set(r.EveCharacter.ToEveEntity())

			sw := border[1].(*kxwidget.Switch)
			sw.On = r.IsTrainingWatched
			sw.Refresh()
			sw.OnChanged = func(on bool) {
				a.updateCharacterWatched(context.Background(), id, on)
			}
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.characters) {
			return
		}
		c := a.characters[id]
		v := !c.IsTrainingWatched
		a.updateCharacterWatched(context.Background(), id, v)
	}
	return l
}

func (a *training) updateCharacterWatched(ctx context.Context, id int, on bool) {
	if id >= len(a.characters) {
		return
	}
	c := a.characters[id]
	runAsync(func() {
		err := a.cw.u.Character().UpdateIsTrainingWatched(ctx, c.ID, on)
		if err != nil {
			slog.Error("Failed to update training watcher", "characterID", c.ID, "error", err)
			a.cw.displaySnackbar("Failed to update training watcher: " + a.cw.u.ErrorDisplay(err))
		}
		fyne.Do(func() {
			// The list may have been reloaded while storing.
			id := a.indexOf(c.ID)
			if id == -1 {
				return
			}
			if err == nil {
				a.characters[id].IsTrainingWatched = on
			}
			a.list.RefreshItem(id) // on error reverts the switch the user already toggled
		})
		if err != nil {
			return
		}
		a.cw.u.Signals().CharacterChanged.Emit(ctx, c.ID)
	})
}

// indexOf returns the row of a character or -1 if not found.
// Must be called on the main thread.
func (a *training) indexOf(characterID int64) int {
	return slices.IndexFunc(a.characters, func(x *app.Character) bool {
		return x.ID == characterID
	})
}

func (a *training) update(ctx context.Context) {
	load := a.charactersLoad.Add(1)
	characters, err := a.cw.u.Character().ListCharacters(ctx)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		if load != a.charactersLoad.Load() {
			return // outdated
		}
		a.cw.reportError("Failed to update training", err)
		return
	}
	slices.SortFunc(characters, func(a, b *app.Character) int {
		return strings.Compare(a.EveCharacter.Name, b.EveCharacter.Name)
	})
	fyne.Do(func() {
		if load != a.charactersLoad.Load() {
			return // outdated
		}
		a.characters = characters
		a.list.Refresh()
	})
}
