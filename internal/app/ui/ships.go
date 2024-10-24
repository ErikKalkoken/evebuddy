package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

// shipsArea is the UI area that shows the skillqueue
type shipsArea struct {
	content       *fyne.Container
	grid          *widget.GridWrap
	groupSelect   *widget.Select
	searchBox     *widget.Entry
	selectedGroup string
	ships         []*app.CharacterShipAbility
	top           *widget.Label
	u             *UI
}

func (u *UI) newShipArea() *shipsArea {
	a := shipsArea{
		ships: make([]*app.CharacterShipAbility, 0),
		top:   widget.NewLabel(""),
		u:     u,
	}
	a.top.TextStyle.Bold = true
	a.searchBox = a.makeSearchBox()
	a.groupSelect = a.makeGroupSelect()
	a.grid = a.makeShipsGrid()
	b := widget.NewButton("Reset", func() {
		a.searchBox.SetText("")
		a.groupSelect.ClearSelected()
	})
	top := container.NewHBox(a.top, layout.NewSpacer(), b)
	entries := container.NewBorder(nil, nil, nil, a.groupSelect, a.searchBox)
	topBox := container.NewVBox(top, widget.NewSeparator(), entries)
	a.content = container.NewBorder(topBox, nil, nil, nil, a.grid)
	return &a
}

func (a *shipsArea) makeGroupSelect() *widget.Select {
	groupSelect := widget.NewSelect([]string{}, func(s string) {
		a.selectedGroup = s
		if err := a.updateEntries(context.TODO()); err != nil {
			t := "Failed to update ship search"
			slog.Error(t, "err", err)
			a.u.statusBarArea.SetError(t)
		}
		a.grid.Refresh()
		a.grid.ScrollToTop()
	})
	return groupSelect
}

func (a *shipsArea) makeSearchBox() *widget.Entry {
	sb := widget.NewEntry()
	sb.SetPlaceHolder("Filter by ship name")
	sb.OnChanged = func(s string) {
		if len(s) == 1 {
			return
		}
		if err := a.updateEntries(context.TODO()); err != nil {
			t := "Failed to update ship search"
			slog.Error(t, "err", err)
			a.u.statusBarArea.SetError(t)
		}
		a.grid.Refresh()
		a.grid.ScrollToTop()
	}
	return sb
}

func (a *shipsArea) makeShipsGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.ships)
		},
		func() fyne.CanvasObject {
			return widgets.NewShipItem(a.u.EveImageService, resourceQuestionmarkSvg)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.ships) {
				return
			}
			o := a.ships[id]
			item := co.(*widgets.ShipItem)
			item.Set(o.Type.ID, o.Type.Name, o.CanFly)
		})
	g.OnSelected = func(id widget.GridWrapItemID) {
		defer g.UnselectAll()
		if id >= len(a.ships) {
			return
		}
		o := a.ships[id]
		a.u.showTypeInfoWindow(o.Type.ID, a.u.characterID())
	}
	return g
}

func (a *shipsArea) refresh() {
	t, i, enabled, err := func() (string, widget.Importance, bool, error) {
		exists := a.u.StatusCacheService.GeneralSectionExists(app.SectionEveCategories)
		if !exists {
			return "Waiting for universe data to be loaded...", widget.WarningImportance, false, nil
		}
		ctx := context.TODO()
		if err := a.updateEntries(ctx); err != nil {
			return "", 0, false, err
		}
		return a.makeTopText(ctx)
	}()
	if err != nil {
		slog.Error("Failed to refresh ships UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.grid.Refresh()
	if enabled {
		a.searchBox.Enable()
	} else {
		a.searchBox.Disable()
	}
}

func (a *shipsArea) updateEntries(ctx context.Context) error {
	if !a.u.hasCharacter() {
		a.ships = make([]*app.CharacterShipAbility, 0)
		a.grid.Refresh()
		a.searchBox.SetText("")
		a.groupSelect.SetOptions([]string{})
		return nil
	}
	characterID := a.u.characterID()
	search := fmt.Sprintf("%%%s%%", a.searchBox.Text)
	oo, err := a.u.CharacterService.ListCharacterShipsAbilities(ctx, characterID, search)
	if err != nil {
		return err
	}
	ships := make([]*app.CharacterShipAbility, 0)
	for _, o := range oo {
		if a.selectedGroup == "" || o.Group.Name == a.selectedGroup {
			ships = append(ships, o)
		}
	}
	a.ships = ships
	a.grid.Refresh()
	groups := set.New[string]()
	for _, o := range oo {
		groups.Add(o.Group.Name)
	}
	g := groups.ToSlice()
	slices.Sort(g)
	a.groupSelect.SetOptions(g)
	return nil
}

func (a *shipsArea) makeTopText(ctx context.Context) (string, widget.Importance, bool, error) {
	if !a.u.hasCharacter() {
		return "No character", widget.LowImportance, false, nil
	}
	characterID := a.u.characterID()
	hasData := a.u.StatusCacheService.CharacterSectionExists(characterID, app.SectionSkills)
	if !hasData {
		return "Waiting for skills to be loaded...", widget.WarningImportance, false, nil
	}
	oo, err := a.u.CharacterService.ListCharacterShipsAbilities(ctx, characterID, "%%")
	if err != nil {
		return "", 0, false, err
	}
	c := 0
	for _, o := range oo {
		if o.CanFly {
			c++
		}
	}
	p := float32(c) / float32(len(oo)) * 100
	text := fmt.Sprintf("Can fly %d / %d ships (%.0f%%)", c, len(oo), p)
	return text, widget.MediumImportance, true, nil
}
