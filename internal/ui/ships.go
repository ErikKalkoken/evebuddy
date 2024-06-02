package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

// shipsArea is the UI area that shows the skillqueue
type shipsArea struct {
	content   *fyne.Container
	entries   binding.UntypedList
	searchBox *widget.Entry
	table     *widget.Table
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
	a.table = a.makeShipsTable()
	topBox := container.NewVBox(a.top, widget.NewSeparator(), a.searchBox)
	a.content = container.NewBorder(topBox, nil, nil, nil, a.table)
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
		a.table.Refresh()
		a.table.ScrollToTop()
	}
	return sb
}

func (a *shipsArea) makeShipsTable() *widget.Table {
	var headers = []struct {
		text  string
		width float32
	}{
		{"Name", 250},
		{"Group", 250},
		{"Can Fly", 50},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return a.entries.Length(), len(headers)
		},
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
			icon.FillMode = canvas.ImageFillOriginal
			label := widget.NewLabel("Template")
			return container.NewHBox(icon, label)
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			icon := row.Objects[0].(*canvas.Image)
			label := row.Objects[1].(*widget.Label)
			label.Importance = widget.MediumImportance
			o, err := getItemUntypedList[*model.CharacterShipAbility](a.entries, tci.Row)
			if err != nil {
				slog.Error("Failed to render ship item in UI", "err", err)
				label.Importance = widget.DangerImportance
				label.Text = "ERROR"
				label.Refresh()
				return
			}
			switch tci.Col {
			case 0:
				refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
					return a.ui.imageManager.InventoryTypeIcon(o.Type.ID, defaultIconSize)
				})
				icon.Show()
				label.Text = o.Type.Name
				label.Show()
			case 1:
				label.Text = o.Group.Name
				icon.Hide()
				label.Show()
			case 2:
				if o.CanFly {
					icon.Resource = theme.NewSuccessThemedResource(theme.ConfirmIcon())
				} else {
					icon.Resource = theme.NewErrorThemedResource(theme.CancelIcon())
				}
				icon.Refresh()
				icon.Show()
				label.Hide()
			}
			label.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		co.(*widget.Label).SetText(s.text)
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.width)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		typ, err := func() (*model.EveType, error) {
			o, err := getItemUntypedList[*model.CharacterShipAbility](a.entries, tci.Row)
			if err != nil {
				return nil, err
			}
			typ, err := a.ui.sv.EveUniverse.GetEveType(context.Background(), o.Type.ID)
			if err != nil {
				return nil, err
			}
			return typ, nil
		}()
		if err != nil {
			t := "Failed to select ship entry"
			slog.Error(t, "err", err)
			a.ui.statusBarArea.SetError(t)
			return
		}
		d := makeTypeDetailDialog(typ.Name, typ.DescriptionPlain(), a.ui.window)
		d.Show()
		t.UnselectAll()
	}
	return t
}

func (a *shipsArea) refresh() {
	t, i, enabled, err := func() (string, widget.Importance, bool, error) {
		_, ok, err := a.ui.sv.Dictionary.GetTime(eveCategoriesKeyLastUpdated)
		if err != nil {
			return "", 0, false, err
		}
		if !ok {
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
	a.table.Refresh()
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
	characterID := a.ui.currentCharID()
	search := fmt.Sprintf("%%%s%%", a.searchBox.Text)
	oo, err := a.ui.sv.Characters.ListCharacterShipsAbilities(context.Background(), characterID, search)
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
	characterID := a.ui.currentCharID()
	ok, err := a.ui.sv.Characters.CharacterSectionWasUpdated(ctx, characterID, model.CharacterSectionSkills)
	if err != nil {
		return "", 0, false, err
	}
	if !ok {
		return "Waiting for skills to be loaded...", widget.WarningImportance, false, nil
	}
	oo, err := a.ui.sv.Characters.ListCharacterShipsAbilities(ctx, characterID, "%%")
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
