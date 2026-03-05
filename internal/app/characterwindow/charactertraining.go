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
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// characterTraining is a UI component that allows to configure training watchers for characters.
type characterTraining struct {
	widget.BaseWidget

	characters []*app.Character
	list       *widget.List
	mc         *manageCharacters
}

func newCharacterTraining(mc *manageCharacters) *characterTraining {
	a := &characterTraining{
		mc: mc,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeList()

	// Signals
	a.mc.signals.CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.mc.signals.CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
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
					d, err := a.mc.cs.TotalTrainingTime(ctx, c.ID)
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
	ab := iwidget.NewAppBar("Watched Training", a.list, actions)
	ab.HideBackground = !a.mc.isMobile
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
				awidget.NewEntityListItem(true, a.mc.eis.CharacterPortraitAsync),
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
		err := a.mc.cs.UpdateIsTrainingWatched(ctx, c.ID, on)
		if err != nil {
			slog.Error("Failed to update training watcher", "characterID", c.ID, "error", err)
			a.mc.sb.Show("Failed to update training watcher: " + a.mc.u.HumanizeError(err))
		}
		fyne.Do(func() {
			a.characters[id].IsTrainingWatched = on
			a.list.RefreshItem(id)
		})
		a.mc.signals.CharacterChanged.Emit(ctx, c.ID)
	}()
}

func (a *characterTraining) update(ctx context.Context) {
	characters, err := a.mc.cs.ListCharacters(ctx)
	if err != nil {
		a.mc.reportError("Failed to update training", err)
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
