package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/widgets"
)

// shipsArea is the UI area that shows the skillqueue
type shipsArea struct {
	content       *fyne.Container
	entries       binding.UntypedList
	searchBox     *widget.Entry
	groupSelect   *widget.Select
	selectedGroup string
	grid          *widget.GridWrap
	top           *widget.Label
	ui            *ui
}

func (u *ui) newShipArea() *shipsArea {
	a := shipsArea{
		ui:      u,
		entries: binding.NewUntypedList(),
		top:     widget.NewLabel(""),
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
		if err := a.updateEntries(); err != nil {
			t := "Failed to update ship search"
			slog.Error(t, "err", err)
			a.ui.statusBarArea.SetError(t)
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
		if err := a.updateEntries(); err != nil {
			t := "Failed to update ship search"
			slog.Error(t, "err", err)
			a.ui.statusBarArea.SetError(t)
		}
		a.grid.Refresh()
		a.grid.ScrollToTop()
	}
	return sb
}

func (a *shipsArea) makeShipsGrid() *widget.GridWrap {
	g := widget.NewGridWrapWithData(
		a.entries,
		func() fyne.CanvasObject {
			return widgets.NewShipItem(a.ui.EveImageService, resourceQuestionmarkSvg)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			item := co.(*widgets.ShipItem)
			o, err := convertDataItem[*app.CharacterShipAbility](di)
			if err != nil {
				slog.Error("Failed to render ship item in UI", "err", err)
				// label.Importance = widget.DangerImportance
				// label.Text = "ERROR"
				// label.Refresh()
				return
			}
			item.Set(o.Type.ID, o.Type.Name, o.CanFly)
		})
	g.OnSelected = func(id widget.GridWrapItemID) {
		defer g.UnselectAll()
		o, err := getItemUntypedList[*app.CharacterShipAbility](a.entries, id)
		if err != nil {
			slog.Error("Failed to select ship", "err", err)
			return
		}
		a.ui.showTypeInfoWindow(o.Type.ID, a.ui.characterID())
	}
	return g
}

func (a *shipsArea) refresh() {
	t, i, enabled, err := func() (string, widget.Importance, bool, error) {
		exists := a.ui.StatusCacheService.GeneralSectionExists(app.SectionEveCategories)
		if !exists {
			return "Waiting for universe data to be loaded...", widget.WarningImportance, false, nil
		}
		if err := a.updateEntries(); err != nil {
			return "", 0, false, err
		}
		return a.makeTopText()
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

func (a *shipsArea) updateEntries() error {
	if !a.ui.hasCharacter() {
		oo := make([]*app.CharacterShipAbility, 0)
		a.entries.Set(copyToUntypedSlice(oo))
		a.searchBox.SetText("")
		a.groupSelect.SetOptions([]string{})
		return nil
	}
	characterID := a.ui.characterID()
	search := fmt.Sprintf("%%%s%%", a.searchBox.Text)
	oo, err := a.ui.CharacterService.ListCharacterShipsAbilities(context.Background(), characterID, search)
	if err != nil {
		return err
	}
	oo2 := make([]*app.CharacterShipAbility, 0)
	for _, o := range oo {
		if a.selectedGroup == "" || o.Group.Name == a.selectedGroup {
			oo2 = append(oo2, o)
		}
	}
	a.entries.Set(copyToUntypedSlice(oo2))
	groups := set.New[string]()
	for _, o := range oo {
		groups.Add(o.Group.Name)
	}
	g := groups.ToSlice()
	slices.Sort(g)
	a.groupSelect.SetOptions(g)
	return nil
}

func (a *shipsArea) makeTopText() (string, widget.Importance, bool, error) {
	ctx := context.Background()
	if !a.ui.hasCharacter() {
		return "No character", widget.LowImportance, false, nil
	}
	characterID := a.ui.characterID()
	hasData := a.ui.StatusCacheService.CharacterSectionExists(characterID, app.SectionSkills)
	if !hasData {
		return "Waiting for skills to be loaded...", widget.WarningImportance, false, nil
	}
	oo, err := a.ui.CharacterService.ListCharacterShipsAbilities(ctx, characterID, "%%")
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
