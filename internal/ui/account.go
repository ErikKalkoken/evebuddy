package ui

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

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
	"github.com/dustin/go-humanize"
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
	bottom     *widget.Label
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
		ui:         u,
	}

	a.bottom = widget.NewLabel("Hint: Click any character to enable it")
	a.bottom.Importance = widget.LowImportance
	a.bottom.Hide()

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
			r, err := u.imageManager.CharacterPortrait(c.id, defaultIconSize)
			if err != nil {
				panic(err)
			}
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
		c, err := getFromBoundUntypedList[accountCharacter](a.characters, id)
		if err != nil {
			slog.Error("failed to access account character in list", "err", err)
			return
		}
		if err := a.ui.LoadCurrentCharacter(c.id); err != nil {
			slog.Error("failed to load current character", "char", c, "err", err)
			return
		}
		a.dialog.Hide()
	}

	b := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		a.showAddCharacterDialog()
	})
	b.Importance = widget.HighImportance
	a.content = container.NewBorder(b, a.bottom, nil, nil, container.NewScroll(list))
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
	if a.characters.Length() > 0 {
		a.bottom.Show()
	} else {
		a.bottom.Hide()
	}
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
	errorMessage  string
	lastUpdatedAt sql.NullTime
	section       string
	timeout       time.Duration
}

func (s *updateStatus) IsOK() bool {
	return s.errorMessage == ""
}

func (a *accountArea) showStatusDialog(c accountCharacter) {
	content := a.makeCharacterStatus(c)
	d1 := dialog.NewCustom("Character update status", "Close", content, a.ui.window)
	d1.Show()
	d1.Resize(fyne.Size{Width: 800, Height: 500})
}

func (a *accountArea) makeCharacterStatus(c accountCharacter) fyne.CanvasObject {
	oo, err := a.ui.service.ListCharacterUpdateStatus(c.id)
	if err != nil {
		panic(err)
	}
	m := make(map[model.CharacterSection]*model.CharacterUpdateStatus)
	for _, o := range oo {
		m[o.Section] = o
	}
	data := make([]updateStatus, len(model.CharacterSections))
	for i, s := range model.CharacterSections {
		x := updateStatus{section: s.Name(), timeout: s.Timeout()}
		o, ok := m[s]
		if ok {
			x.lastUpdatedAt = o.LastUpdatedAt
			x.lastUpdatedAt.Valid = true
		}
		data[i] = x
	}
	var headers = []struct {
		text  string
		width float32
	}{
		{"Section", 150},
		{"Timeout", 150},
		{"Last Update", 150},
		{"Status", 150},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(data), len(headers)

		},
		func() fyne.CanvasObject {
			l := widget.NewLabel("Placeholder")
			l.Truncation = fyne.TextTruncateEllipsis
			return l
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			d := data[tci.Row]
			label := co.(*widget.Label)
			var s string
			i := widget.MediumImportance
			switch tci.Col {
			case 0:
				s = d.section
			case 1:
				now := time.Now()
				s = humanize.RelTime(now.Add(d.timeout), now, "", "")
			case 2:
				s = humanizedNullTime(d.lastUpdatedAt, "?")
			case 3:
				if d.IsOK() {
					s = "OK"
					i = widget.SuccessImportance
				} else {
					s = d.errorMessage
					i = widget.DangerImportance
				}
			}
			label.Text = s
			label.Importance = i
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
	t.OnSelected = func(id widget.TableCellID) {
		t.UnselectAll()
	}

	top := widget.NewLabel(fmt.Sprintf("Update status for %s", c.name))
	top.TextStyle.Bold = true
	return container.NewBorder(top, nil, nil, nil, t)
}
