package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/dustin/go-humanize"
)

// shipsArea is the UI area that shows the skillqueue
type shipsArea struct {
	content *fyne.Container
	entries binding.UntypedList
	table   *widget.Table
	total   *widget.Label
	ui      *ui
}

func (u *ui) NewShipArea() *shipsArea {
	a := shipsArea{
		ui:      u,
		entries: binding.NewUntypedList(),
		total:   widget.NewLabel(""),
	}
	a.total.TextStyle.Bold = true
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
			// label.Truncation = fyne.TextTruncateEllipsis
			return container.NewHBox(icon, label)
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			icon := row.Objects[0].(*canvas.Image)
			label := row.Objects[1].(*widget.Label)
			label.Importance = widget.MediumImportance
			o, err := getFromBoundUntypedList[model.CharacterShipAbility](a.entries, tci.Row)
			if err != nil {
				panic(err)
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

	top := container.NewVBox(a.total, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, t)
	a.table = t
	a.entries.AddListener(binding.NewDataListener(func() {
		a.table.Refresh()
	}))
	return &a
}

func (a *shipsArea) Refresh() {
	canFly, err := a.updateEntries()
	if err != nil {
		panic(err)
	}
	s, i := a.makeTopText(canFly)
	a.total.Text = s
	a.total.Importance = i
	a.total.Refresh()
}

func (a *shipsArea) makeTopText(canFly int) (string, widget.Importance) {
	c := a.ui.CurrentChar()
	if c == nil {
		return "No data yet...", widget.LowImportance
	}
	hasData, err := a.ui.service.CharacterSectionWasUpdated(c.ID, model.CharacterSectionWalletJournal)
	if err != nil {
		return "ERROR", widget.DangerImportance
	}
	if !hasData {
		return "No data yet...", widget.LowImportance
	}
	total := a.entries.Length()
	p := float32(canFly) / float32(total) * 100
	s := fmt.Sprintf("Count: %s (%s)", humanize.Comma(int64(total)), fmt.Sprintf("%.0f%%", p))
	return s, widget.MediumImportance
}

func (a *shipsArea) updateEntries() (int, error) {
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		oo := make([]model.CharacterShipAbility, 0)
		a.entries.Set(copyToUntypedSlice(oo))
	}
	oo, err := a.ui.service.ListCharacterShipsAbilities(characterID)
	if err != nil {
		return 0, err
	}
	c := 0
	for _, o := range oo {
		if o.CanFly {
			c++
		}
	}
	a.entries.Set(copyToUntypedSlice(oo))
	return c, nil
}
