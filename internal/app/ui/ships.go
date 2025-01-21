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

const (
	flyableCan    = "Can Fly"
	flyableCanNot = "Can Not Fly"
)

// ShipsArea is the UI area that shows the skillqueue
type ShipsArea struct {
	Content         *fyne.Container
	flyableSelect   *widget.Select
	flyableSelected string
	grid            *widget.GridWrap
	groupSelect     *widget.Select
	groupSelected   string
	searchBox       *widget.Entry
	ships           []*app.CharacterShipAbility
	top             *widget.Label
	foundText       *widget.Label
	u               *BaseUI
}

func (u *BaseUI) newShipArea() *ShipsArea {
	a := ShipsArea{
		ships:     make([]*app.CharacterShipAbility, 0),
		top:       widget.NewLabel(""),
		foundText: widget.NewLabel(""),
		u:         u,
	}
	a.top.TextStyle.Bold = true

	a.searchBox = widget.NewEntry()
	a.searchBox.SetPlaceHolder("Filter by ship name")
	a.searchBox.OnChanged = func(s string) {
		if len(s) == 1 {
			return
		}
		if err := a.updateEntries(); err != nil {
			d := NewErrorDialog("Failed to update ships", err, a.u.Window)
			d.Show()
		}
		a.grid.Refresh()
		a.grid.ScrollToTop()
	}

	a.groupSelect = widget.NewSelect([]string{}, func(s string) {
		a.groupSelected = s
		if err := a.updateEntries(); err != nil {
			d := NewErrorDialog("Failed to update ships", err, a.u.Window)
			d.Show()
		}
		a.grid.Refresh()
		a.grid.ScrollToTop()
	})
	a.groupSelect.PlaceHolder = "(Ship Class)"

	a.flyableSelect = widget.NewSelect([]string{}, func(s string) {
		a.flyableSelected = s
		if err := a.updateEntries(); err != nil {
			d := NewErrorDialog("Failed to update ships", err, a.u.Window)
			d.Show()
		}
		a.grid.Refresh()
		a.grid.ScrollToTop()
	})
	a.flyableSelect.PlaceHolder = "(Flyable)"

	a.grid = a.makeShipsGrid()
	b := widget.NewButton("Reset", func() {
		a.reset()
	})
	top := container.NewHBox(a.top, a.foundText, layout.NewSpacer(), b)
	entries := container.NewBorder(nil, nil, nil, container.NewHBox(a.groupSelect, a.flyableSelect), a.searchBox)
	topBox := container.NewVBox(top, widget.NewSeparator(), entries)
	a.Content = container.NewBorder(topBox, nil, nil, nil, a.grid)
	return &a
}

func (a *ShipsArea) reset() {
	a.searchBox.SetText("")
	a.groupSelect.ClearSelected()
	a.flyableSelect.ClearSelected()
	a.foundText.Hide()
}

func (a *ShipsArea) makeShipsGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.ships)
		},
		func() fyne.CanvasObject {
			return widgets.NewShipItem(a.u.EveImageService, a.u.CacheService, IconQuestionmarkSvg)
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
		a.u.ShowTypeInfoWindow(o.Type.ID, a.u.CharacterID(), RequirementsTab)
	}
	return g
}

func (a *ShipsArea) Refresh() {
	t, i, enabled, err := func() (string, widget.Importance, bool, error) {
		exists := a.u.StatusCacheService.GeneralSectionExists(app.SectionEveCategories)
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
	a.reset()
}

func (a *ShipsArea) updateEntries() error {
	if !a.u.HasCharacter() {
		a.ships = make([]*app.CharacterShipAbility, 0)
		a.grid.Refresh()
		a.searchBox.SetText("")
		a.groupSelect.SetOptions([]string{})
		a.flyableSelect.SetOptions([]string{})
		return nil
	}
	characterID := a.u.CharacterID()
	search := fmt.Sprintf("%%%s%%", a.searchBox.Text)
	oo, err := a.u.CharacterService.ListCharacterShipsAbilities(context.Background(), characterID, search)
	if err != nil {
		return err
	}
	ships := make([]*app.CharacterShipAbility, 0)
	for _, o := range oo {
		isSelectedGroup := a.groupSelected == "" || o.Group.Name == a.groupSelected
		var isSelectedFlyable bool
		switch a.flyableSelected {
		case flyableCan:
			isSelectedFlyable = o.CanFly
		case flyableCanNot:
			isSelectedFlyable = !o.CanFly
		default:
			isSelectedFlyable = true
		}
		if isSelectedGroup && isSelectedFlyable {
			ships = append(ships, o)
		}
	}
	a.ships = ships
	a.grid.Refresh()
	g := set.New[string]()
	f := set.New[string]()
	for _, o := range ships {
		g.Add(o.Group.Name)
		if o.CanFly {
			f.Add(flyableCan)
		} else {
			f.Add(flyableCanNot)
		}
	}
	groups := g.ToSlice()
	slices.Sort(groups)
	a.groupSelect.SetOptions(groups)
	flyable := f.ToSlice()
	slices.Sort(flyable)
	a.flyableSelect.SetOptions(flyable)
	a.foundText.SetText(fmt.Sprintf("%d found", len(ships)))
	a.foundText.Show()
	return nil
}

func (a *ShipsArea) makeTopText() (string, widget.Importance, bool, error) {
	if !a.u.HasCharacter() {
		return "No character", widget.LowImportance, false, nil
	}
	characterID := a.u.CharacterID()
	hasData := a.u.StatusCacheService.CharacterSectionExists(characterID, app.SectionSkills)
	if !hasData {
		return "Waiting for skills to be loaded...", widget.WarningImportance, false, nil
	}
	oo, err := a.u.CharacterService.ListCharacterShipsAbilities(context.Background(), characterID, "%%")
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
