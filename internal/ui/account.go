package ui

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
)

type accountCharacter struct {
	id   int32
	name string
}

// accountArea is the UI area for managing of characters.
type accountArea struct {
	characters binding.UntypedList
	content    *fyne.Container
	dialog     *dialog.CustomDialog
	total      *widget.Label
	ui         *ui
}

func (u *ui) ShowAccountDialog() {
	a := u.NewAccountArea()
	dialog := dialog.NewCustom("Manage Characters", "Close", a.content, u.window)
	a.dialog = dialog
	dialog.Show()
	dialog.Resize(fyne.Size{Width: 500, Height: 500})
	err := a.Refresh()
	if err != nil {
		u.statusArea.SetError("Failed to open dialog to manage characters")
		dialog.Hide()
	}
}

func (u *ui) NewAccountArea() *accountArea {
	a := &accountArea{
		characters: binding.NewUntypedList(),
		total:      widget.NewLabel(""),
		ui:         u,
	}

	list := widget.NewListWithData(
		a.characters,
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
			icon.FillMode = canvas.ImageFillContain
			icon.SetMinSize(fyne.Size{Width: defaultIconSize, Height: defaultIconSize})
			name := widget.NewLabel("Template")
			b1 := widget.NewButtonWithIcon("Status", theme.ComputerIcon(), func() {})
			b2 := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {})
			b2.Importance = widget.DangerImportance
			row := container.NewHBox(icon, name, layout.NewSpacer(), b1, b2)
			return row

			// hasToken, err := a.ui.service.HasTokenWithScopes(char.ID)
			// if err != nil {
			// 	slog.Error("Can not check if character has token", "err", err)
			// 	continue
			// }
			// if !hasToken {
			// 	row.Add(widget.NewIcon(theme.WarningIcon()))
			// }

		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			row := co.(*fyne.Container)

			name := row.Objects[1].(*widget.Label)
			c, err := convertDataItem[accountCharacter](di)
			if err != nil {
				slog.Error("failed to render row account table", "err", err)
				name.Text = "failed to render"
				name.Importance = widget.DangerImportance
				name.Refresh()
				return
			}
			name.SetText(c.name)

			icon := row.Objects[0].(*canvas.Image)
			r := u.imageManager.CharacterPortrait(c.id, defaultIconSize)
			image := canvas.NewImageFromResource(r)
			icon.Resource = image.Resource
			image.Refresh()

			row.Objects[3].(*widget.Button).OnTapped = func() {
				a.showStatusDialog(c)
			}

			row.Objects[4].(*widget.Button).OnTapped = func() {
				a.showDeleteDialog(c)
			}
		})

	list.OnSelected = func(id widget.ListItemID) {
		list.UnselectAll()
	}

	b := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		a.showAddCharacterDialog()
	})
	b.Importance = widget.HighImportance
	a.content = container.NewBorder(b, a.total, nil, nil, container.NewScroll(list))
	return a
}

func (a *accountArea) showDeleteDialog(c accountCharacter) {
	d1 := dialog.NewConfirm(
		"Delete Character",
		fmt.Sprintf("Are you sure you want to delete %s?", c.name),
		func(confirmed bool) {
			if confirmed {
				err := a.ui.service.DeleteCharacter(c.id)
				if err != nil {
					d2 := dialog.NewError(err, a.ui.window)
					d2.Show()
				}
				if err := a.Refresh(); err != nil {
					panic(err)
				}
				isCurrentChar := c.id == a.ui.CurrentCharID()
				if isCurrentChar {
					err := a.ui.SetAnyCharacter()
					if err != nil {
						panic(err)
					}
				}
				a.ui.RefreshOverview()
				a.ui.toolbarArea.Refresh()
			}
		},
		a.ui.window,
	)
	d1.Show()
}

func (a *accountArea) Refresh() error {
	cc, err := a.ui.service.ListCharactersShort()
	if err != nil {
		return err
	}
	cc2 := make([]accountCharacter, len(cc))
	for i, c := range cc {
		cc2[i] = accountCharacter{id: c.ID, name: c.Name}
	}
	if err := a.characters.Set(copyToUntypedSlice(cc2)); err != nil {
		return err
	}
	a.total.SetText(fmt.Sprintf("Characters: %d", a.characters.Length()))
	return nil
}

func (a *accountArea) showAddCharacterDialog() {
	ctx, cancel := context.WithCancel(context.Background())
	s := "Please follow instructions in your browser to add a new character."
	infoText := binding.BindString(&s)
	content := widget.NewLabelWithData(infoText)
	d1 := dialog.NewCustom(
		"Add Character",
		"Cancel",
		content,
		a.ui.window,
	)
	d1.SetOnClosed(cancel)
	go func() {
		characterID, err := a.ui.service.UpdateOrCreateCharacterFromSSO(ctx, infoText)
		if err != nil {
			if !errors.Is(err, service.ErrAborted) {
				slog.Error("Failed to add a new character", "error", err)
				d2 := dialog.NewInformation(
					"Error",
					fmt.Sprintf("An error occurred when trying to add a new character:\n%s", err),
					a.ui.window,
				)
				d2.Show()
			}
		} else {
			isFirst := a.characters.Length() == 0
			if err := a.Refresh(); err != nil {
				panic(err)
			}
			a.ui.RefreshOverview()
			a.ui.toolbarArea.Refresh()
			if isFirst {
				if err := a.ui.SetAnyCharacter(); err != nil {
					panic(err)
				}
			} else {
				a.ui.overviewArea.MaybeUpdateAndRefresh(characterID)
			}
		}
		d1.Hide()
	}()
	d1.Show()
}

type updateStatus struct {
	section       string
	lastUpdatedAt sql.NullTime
}

func (a *accountArea) showStatusDialog(c accountCharacter) {
	content := a.makeCharacterStatus(c)
	d1 := dialog.NewCustom("Character status", "Close", content, a.ui.window)
	d1.Show()
	d1.Resize(fyne.Size{Width: 600, Height: 600})

}

func (a *accountArea) makeCharacterStatus(c accountCharacter) fyne.CanvasObject {
	oo, err := a.ui.service.ListCharacterUpdateStatus(c.id)
	if err != nil {
		panic(err)
	}
	m := make(map[model.CharacterSection]*model.CharacterUpdateStatus)
	for _, o := range oo {
		m[o.SectionID] = o
	}
	data := make([]updateStatus, len(model.CharacterSections))
	for i, s := range model.CharacterSections {
		x := updateStatus{section: s.Name()}
		o, ok := m[s]
		if ok {
			x.lastUpdatedAt.Time = o.UpdatedAt
			x.lastUpdatedAt.Valid = true
		}
		data[i] = x
	}
	var headers = []struct {
		text  string
		width float32
	}{
		{"Section", 200},
		{"Last Update", 200},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(data), len(headers)

		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Placeholder")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			d := data[tci.Row]
			cell := co.(*widget.Label)
			var s string
			switch tci.Col {
			case 0:
				s = d.section
			case 1:
				s = humanizedNullTime(d.lastUpdatedAt, "?")
			}
			cell.SetText(s)
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
	t.OnSelected = func(id widget.TableCellID) {
		t.UnselectAll()
	}

	top := widget.NewLabel(fmt.Sprintf("Update status for %s", c.name))
	top.TextStyle.Bold = true
	return container.NewBorder(top, nil, nil, nil, t)
}
