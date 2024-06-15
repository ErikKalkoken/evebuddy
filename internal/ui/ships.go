package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

// shipsArea is the UI area that shows the skillqueue
type shipsArea struct {
	content   *fyne.Container
	entries   binding.UntypedList
	searchBox *widget.Entry
	grid      *widget.GridWrap
	top       *widget.Label
	ui        *ui
}

func (u *ui) newShipArea() *shipsArea {
	a := shipsArea{
		ui:      u,
		entries: binding.NewUntypedList(),
		top:     widget.NewLabel(""),
	}
	a.top.TextStyle.Bold = true
	a.searchBox = a.makeSearchBox()
	a.grid = a.makeShipsTable()
	topBox := container.NewVBox(a.top, widget.NewSeparator(), a.searchBox)
	a.content = container.NewBorder(topBox, nil, nil, nil, a.grid)
	return &a
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

func (a *shipsArea) makeShipsTable() *widget.GridWrap {
	t := widget.NewGridWrapWithData(
		a.entries,
		func() fyne.CanvasObject {
			image := canvas.NewImageFromResource(resourceQuestionmarkSvg)
			image.FillMode = canvas.ImageFillContain
			image.SetMinSize(fyne.Size{Width: 128, Height: 128})
			label := widget.NewLabel("First line\nSecond Line\nThird Line")
			return container.NewVBox(image, label)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			icon := row.Objects[0].(*canvas.Image)
			label := row.Objects[1].(*widget.Label)
			o, err := convertDataItem[*model.CharacterShipAbility](di)
			if err != nil {
				slog.Error("Failed to render ship item in UI", "err", err)
				label.Importance = widget.DangerImportance
				label.Text = "ERROR"
				label.Refresh()
				return
			}
			label.Text = o.Type.Name
			label.Wrapping = fyne.TextWrapWord
			var i widget.Importance
			if o.CanFly {
				i = widget.MediumImportance
			} else {
				i = widget.LowImportance
			}
			label.Importance = i
			label.Refresh()
			refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
				return a.ui.sv.EveImage.InventoryTypeRender(o.Type.ID, 256)
			})
			//
			// 	switch tci.Col {
			// 	case 0:
			// 	case 1:
			// 		label.Text = o.Group.Name
			// 		icon.Hide()
			// 		label.Show()
			// 	case 2:
			// 		icon.Resource = boolIconResource(o.CanFly)
			// 		icon.Refresh()
			// 		icon.Show()
			// 		label.Hide()
			// 	}
			// 	label.Refresh()
			// },
		})

	t.OnSelected = func(id widget.GridWrapItemID) {
		defer t.UnselectAll()
		o, err := getItemUntypedList[*model.CharacterShipAbility](a.entries, id)
		if err != nil {
			slog.Error("Failed to select ship", "err", err)
			return
		}
		a.ui.showTypeInfoWindow(o.Type.ID, a.ui.characterID())
	}
	return t
}

func (a *shipsArea) refresh() {
	t, i, enabled, err := func() (string, widget.Importance, bool, error) {
		exists := a.ui.sv.StatusCache.GeneralSectionExists(model.SectionEveCategories)
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
		oo := make([]*model.CharacterShipAbility, 0)
		a.entries.Set(copyToUntypedSlice(oo))
		a.searchBox.SetText("")
		return nil
	}
	characterID := a.ui.characterID()
	search := fmt.Sprintf("%%%s%%", a.searchBox.Text)
	oo, err := a.ui.sv.Character.ListCharacterShipsAbilities(context.Background(), characterID, search)
	if err != nil {
		return err
	}
	a.entries.Set(copyToUntypedSlice(oo))
	return nil
}

func (a *shipsArea) makeTopText() (string, widget.Importance, bool, error) {
	ctx := context.Background()
	if !a.ui.hasCharacter() {
		return "No character", widget.LowImportance, false, nil
	}
	characterID := a.ui.characterID()
	hasData := a.ui.sv.StatusCache.CharacterSectionExists(characterID, model.SectionSkills)
	if !hasData {
		return "Waiting for skills to be loaded...", widget.WarningImportance, false, nil
	}
	oo, err := a.ui.sv.Character.ListCharacterShipsAbilities(ctx, characterID, "%%")
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
