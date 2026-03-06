package characterwindow

import (
	"context"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	awidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// characterTraining is a UI component that allows to configure training watchers for characters.
type characterTraining struct {
	widget.BaseWidget

	characters []*app.Character
	cw         *characterWindow
	list       *widget.List
}

func newCharacterTraining(cw *characterWindow) *characterTraining {
	a := &characterTraining{
		cw: cw,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeList()

	// Signals
	a.cw.signals.CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.cw.signals.CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})
	return a
}

func (a *characterTraining) CreateRenderer() fyne.WidgetRenderer {
	actions := kxwidget.NewIconButtonWithMenu(theme.MoreHorizontalIcon(), fyne.NewMenu("",
		fyne.NewMenuItem("Set to currently trained", func() {
			go func() {
				ctx := context.Background()
				for id, c := range a.characters {
					d, err := a.cw.cs.TotalTrainingTime(ctx, c.ID)
					if err != nil {
						slog.Error("Failed to set watcher for trained characters", "error", err)
						continue
					}
					fyne.Do(func() {
						a.updateCharacterWatched(ctx, id, d.ValueOrZero() > 0)
					})
				}
			}()
		}),
		fyne.NewMenuItem("Enable all", func() {
			ctx := context.Background()
			for id := range a.characters {
				a.updateCharacterWatched(ctx, id, true)
			}
		}),
		fyne.NewMenuItem("Disable all", func() {
			ctx := context.Background()
			for id := range a.characters {
				a.updateCharacterWatched(ctx, id, false)
			}
		}),
	))
	ab := xwidget.NewAppBar("Watched Training", a.list, actions)
	ab.HideBackground = !app.IsMobile()
	return widget.NewSimpleRenderer(ab)
}

func (a *characterTraining) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			return container.NewBorder(
				nil,
				nil,
				nil,
				kxwidget.NewSwitch(nil),
				awidget.NewEntityListItem(true, a.cw.eis.CharacterPortraitAsync),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.characters) {
				return
			}
			r := a.characters[id]
			border := co.(*fyne.Container).Objects

			border[0].(*awidget.EntityListItem).Set(r.EveCharacter.ID, r.EveCharacter.Name)

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

func (a *characterTraining) updateCharacterWatched(ctx context.Context, id int, on bool) {
	if id >= len(a.characters) {
		return
	}
	c := a.characters[id]
	go func() {
		err := a.cw.cs.UpdateIsTrainingWatched(ctx, c.ID, on)
		if err != nil {
			slog.Error("Failed to update training watcher", "characterID", c.ID, "error", err)
			a.cw.sb.Show("Failed to update training watcher: " + app.ErrorDisplay(err))
		}
		fyne.Do(func() {
			a.characters[id].IsTrainingWatched = on
			a.list.RefreshItem(id)
		})
		a.cw.signals.CharacterChanged.Emit(ctx, c.ID)
	}()
}

func (a *characterTraining) update(ctx context.Context) {
	characters, err := a.cw.cs.ListCharacters(ctx)
	if err != nil {
		a.cw.reportError("Failed to update training", err)
		return
	}
	slices.SortFunc(characters, func(a, b *app.Character) int {
		return strings.Compare(a.EveCharacter.Name, b.EveCharacter.Name)
	})
	fyne.Do(func() {
		a.characters = characters
		a.list.Refresh()
	})
}
