package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const labelWith = 45

type characterSendMail struct {
	widget.BaseWidget

	body      *widget.Entry
	character atomic.Pointer[app.Character]
	from      *eveEntityEntry
	subject   *widget.Entry
	to        *eveEntityEntry
	u         *baseUI
	w         fyne.Window
}

func newCharacterSendMail(u *baseUI, c *app.Character, mode app.SendMailMode, m *app.CharacterMail) *characterSendMail {
	a := &characterSendMail{
		u: u,
		w: u.MainWindow(),
	}
	a.character.Store(c)
	a.ExtendBaseWidget(a)

	a.from = newEveEntityEntry(widget.NewLabel("From"), labelWith, u.eis)
	a.from.ShowInfoWindow = u.ShowEveEntityInfoWindow
	a.from.Set([]*app.EveEntity{{ID: c.ID, Name: c.EveCharacter.Name, Category: app.EveEntityCharacter}})
	a.from.Disable()

	toButton := widget.NewButton("To", func() {
		showAddDialog(u, c.ID, func(ee *app.EveEntity) {
			a.to.Add(ee)
		}, a.w)
	})
	a.to = newEveEntityEntry(toButton, labelWith, u.eis)
	a.to.ShowInfoWindow = u.ShowEveEntityInfoWindow
	a.to.Placeholder = "Tap To-Button to add recipients..."

	a.subject = widget.NewEntry()
	a.subject.PlaceHolder = "Subject"

	a.body = widget.NewEntry()
	a.body.MultiLine = true
	a.body.Wrapping = fyne.TextWrapWord
	a.body.SetMinRowsVisible(14)
	a.body.PlaceHolder = "Compose message"

	if m != nil {
		const sep = "\n\n--------------------------------\n"
		switch mode {
		case app.SendMailReply:
			a.to.Set([]*app.EveEntity{m.From})
			a.subject.SetText(fmt.Sprintf("Re: %s", m.Subject))
			a.body.SetText(sep + m.String())
		case app.SendMailReplyAll:
			oo := slices.Concat([]*app.EveEntity{m.From}, m.Recipients)
			oo = slices.DeleteFunc(oo, func(o *app.EveEntity) bool {
				return o.ID == c.EveCharacter.ID
			})
			a.to.Set(oo)
			a.subject.SetText(fmt.Sprintf("Re: %s", m.Subject))
			a.body.SetText(sep + m.String())
		case app.SendMailForward:
			a.subject.SetText(fmt.Sprintf("Fw: %s", m.Subject))
			a.body.SetText(sep + m.String())
		default:
			panic(fmt.Errorf("undefined mode for create message: %v", mode))
		}
	}
	return a
}

func (a *characterSendMail) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(a.from, a.to, a.subject),
		nil,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *characterSendMail) SetWindow(w fyne.Window) {
	a.w = w
}

// sendAction tries to send the current mail and reports whether it was successful
func (a *characterSendMail) SendAction() bool {
	showErrorDialog := func(message string) {
		a.u.ShowInformationDialog("Failed to send mail", message, a.u.MainWindow())
	}
	if a.to.IsEmpty() {
		showErrorDialog("A mail needs to have at least one recipient.")
		return false
	}
	if a.subject.Text == "" {
		showErrorDialog("The subject can not be empty.")
		return false
	}
	if a.body.Text == "" {
		showErrorDialog("The message can not be empty.")
		return false
	}
	ctx := context.Background()
	c := a.character.Load()
	_, err := a.u.cs.SendMail(
		ctx,
		c.ID,
		a.subject.Text,
		a.to.Items(),
		a.body.Text,
	)
	if err != nil {
		showErrorDialog(err.Error())
		return false
	}
	a.u.characterSectionChanged.Emit(ctx, characterSectionUpdated{
		characterID: c.ID,
		section:     app.SectionCharacterMailHeaders,
	})
	a.u.ShowSnackbar(fmt.Sprintf("Your mail to %s has been sent.", a.to))
	return true
}

func showAddDialog(u *baseUI, characterID int32, onSelected func(ee *app.EveEntity), w fyne.Window) {
	var modal *widget.PopUp
	results := make([]*app.EveEntity, 0)
	fallbackIcon := icons.Questionmark32Png
	list := widget.NewList(
		func() int {
			return len(results)
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("Template")
			name.Truncation = fyne.TextTruncateClip
			category := widget.NewLabel("Template")
			category.SizeName = theme.SizeNameCaptionText
			icon := iwidget.NewImageFromResource(icons.Questionmark32Png, fyne.NewSquareSize(app.IconUnitSize))
			return container.NewBorder(
				nil,
				nil,
				icon,
				category,
				name,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(results) {
				return
			}
			ee := results[id]
			row := co.(*fyne.Container).Objects
			row[0].(*widget.Label).SetText(ee.Name)
			image := row[1].(*canvas.Image)
			iwidget.RefreshImageAsync(image, func() (fyne.Resource, error) {
				res, err := fetchEveEntityAvatar(u.eis, ee, fallbackIcon)
				if err != nil {
					return fallbackIcon, err
				}
				return res, nil
			})
			row[2].(*widget.Label).SetText(ee.CategoryDisplay())
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(results) {
			list.UnselectAll()
			return
		}
		onSelected(results[id])
		modal.Hide()
	}
	showErrorDialog := func(search string, err error) {
		slog.Error("Failed to resolve names", "search", search, "error", err)
		u.showErrorDialog("Something went wrong", err, w)
	}
	entry := widget.NewEntry()
	entry.PlaceHolder = "Type to start searching..."
	entry.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		entry.SetText("")
	})
	entry.OnChanged = func(search string) {
		if len(search) < 3 {
			results = results[:0]
			list.Refresh()
			return
		}
		go func() {
			var err error
			results, err = u.eus.ListEntitiesByPartialName(context.Background(), search)
			if err != nil {
				fyne.Do(func() {
					showErrorDialog(search, err)
				})
				return
			}
			fyne.Do(func() {
				list.Refresh()
			})
		}()
		go func() {
			ctx := context.Background()
			missingIDs, err := u.cs.AddEveEntitiesFromSearchESI(
				ctx,
				characterID,
				search,
			)
			if err != nil {
				fyne.Do(func() {
					showErrorDialog(search, err)
				})
				return
			}
			if missingIDs.Size() == 0 {
				return // no need to update when not changed
			}
			results, err = u.eus.ListEntitiesByPartialName(ctx, search)
			if err != nil {
				fyne.Do(func() {
					showErrorDialog(search, err)
				})
				return
			}
			fyne.Do(func() {
				list.Refresh()
			})
		}()
	}
	c := container.NewBorder(
		container.NewBorder(
			widget.NewLabel("Add Recipient"),
			nil,
			nil,
			widget.NewButton("Cancel", func() {
				modal.Hide()
			}),
			entry,
		),
		nil,
		nil,
		nil,
		list,
	)
	modal = widget.NewModalPopUp(c, w.Canvas())
	_, s := w.Canvas().InteractiveArea()
	modal.Resize(fyne.NewSize(s.Width, s.Height))
	modal.Show()
	w.Canvas().Focus(entry)
}
