package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type SendMailMode uint

const (
	SendMailNew SendMailMode = iota + 1
	SendMailReply
	SendMailReplyAll
	SendMailForward
)

func (u *BaseUI) MakeSendMailPage(
	character *app.Character,
	mode SendMailMode,
	mail *app.CharacterMail,
	w fyne.Window,
) (fyne.CanvasObject, fyne.Resource, func() bool) {
	const labelWith = 45

	from := appwidget.NewEveEntityEntry(widget.NewLabel("From"), labelWith, u.EveImageService)
	from.Set([]*app.EveEntity{{ID: character.ID, Name: character.EveCharacter.Name, Category: app.EveEntityCharacter}})
	from.Disable()

	var to *appwidget.EveEntityEntry
	toButton := widget.NewButton("To", func() {
		u.showAddDialog(character.ID, func(ee *app.EveEntity) {
			to.Add(ee)
		}, w)
	})
	to = appwidget.NewEveEntityEntry(toButton, labelWith, u.EveImageService)
	to.Placeholder = "Tap To-Button to add recipients..."

	subject := widget.NewEntry()
	subject.PlaceHolder = "Subject"

	body := widget.NewEntry()
	body.MultiLine = true
	body.SetMinRowsVisible(14)
	body.PlaceHolder = "Compose message"

	if mail != nil {
		const sep = "\n\n--------------------------------\n"
		switch mode {
		case SendMailReply:
			to.Set([]*app.EveEntity{mail.From})
			subject.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			body.SetText(sep + mail.String())
		case SendMailReplyAll:
			oo := slices.Concat([]*app.EveEntity{mail.From}, mail.Recipients)
			oo = slices.DeleteFunc(oo, func(o *app.EveEntity) bool {
				return o.ID == character.EveCharacter.ID
			})
			to.Set(oo)
			subject.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			body.SetText(sep + mail.String())
		case SendMailForward:
			subject.SetText(fmt.Sprintf("Fw: %s", mail.Subject))
			body.SetText(sep + mail.String())
		default:
			panic(fmt.Errorf("undefined mode for create message: %v", mode))
		}
	}

	// sendAction tries to send the current mail and reports whether it was successful
	sendAction := func() bool {
		showErrorDialog := func(message string) {
			d := dialog.NewInformation("Failed to send mail", message, w)
			d.Show()
		}
		if to.IsEmpty() {
			showErrorDialog("A mail needs to have at least one recipient.")
			return false
		}
		if subject.Text == "" {
			showErrorDialog("The subject can not be empty.")
			return false
		}
		if body.Text == "" {
			showErrorDialog("The message can not be empty.")
			return false
		}
		ctx := context.Background()
		_, err := u.CharacterService.SendCharacterMail(
			ctx,
			character.ID,
			subject.Text,
			to.Items(),
			body.Text,
		)
		if err != nil {
			showErrorDialog(err.Error())
			return false
		}
		u.Snackbar.Show("Your mail has been sent.")
		return true
	}
	page := container.NewBorder(
		container.NewVBox(from, to, subject),
		nil,
		nil,
		nil,
		body,
	)
	return page, theme.MailSendIcon(), sendAction
}

func (u *BaseUI) showAddDialog(characterID int32, onSelected func(ee *app.EveEntity), w fyne.Window) {
	var modal *widget.PopUp
	items := make([]*app.EveEntity, 0)
	list := widget.NewList(
		func() int {
			return len(items)
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("Template")
			name.Truncation = fyne.TextTruncateClip
			category := iwidget.NewLabelWithSize("Template", theme.SizeNameCaptionText)
			icon := iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(DefaultIconUnitSize))
			return container.NewBorder(
				nil,
				nil,
				icon,
				category,
				name,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(items) {
				return
			}
			ee := items[id]
			row := co.(*fyne.Container).Objects
			row[0].(*widget.Label).SetText(ee.Name)
			image := row[1].(*canvas.Image)
			RefreshImageResourceAsync(image, func() (fyne.Resource, error) {
				res, err := u.EveImageService.EntityIcon(ee.ID, ee.Category.ToEveImage(), DefaultIconPixelSize)
				if err != nil {
					slog.Error("eve entity entry icon update", "error", err)
					res = icon.Questionmark32Png
				}
				res, err = fynetools.MakeAvatar(res)
				if err != nil {
					slog.Error("eve entity entry make avatar", "error", err)
					res = icon.Questionmark32Png
				}
				return res, nil
			})
			row[2].(*iwidget.Label).SetText(ee.CategoryDisplay())
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(items) {
			list.UnselectAll()
			return
		}
		onSelected(items[id])
		modal.Hide()
	}
	showErrorDialog := func(search string, err error) {
		slog.Error("Failed to resolve names", "search", search, "error", err)
		d := dialog.NewInformation("Something went wrong", err.Error(), w)
		d.Show()
	}
	entry := widget.NewEntry()
	entry.PlaceHolder = "Type to start searching..."
	entry.ActionItem = iwidget.NewIconButton(theme.CancelIcon(), func() {
		entry.SetText("")
	})
	entry.OnChanged = func(search string) {
		if len(search) < 3 {
			items = items[:0]
			list.Refresh()
			return
		}
		ctx := context.Background()
		var err error
		items, err = u.EveUniverseService.ListEveEntitiesByPartialName(ctx, search)
		if err != nil {
			showErrorDialog(search, err)
			return
		}
		list.Refresh()
		go func() {
			missingIDs, err := u.CharacterService.AddEveEntitiesFromCharacterSearchESI(
				ctx,
				characterID,
				search,
			)
			if err != nil {
				showErrorDialog(search, err)
				return
			}
			if len(missingIDs) == 0 {
				return // no need to update when not changed
			}
			items, err = u.EveUniverseService.ListEveEntitiesByPartialName(ctx, search)
			if err != nil {
				showErrorDialog(search, err)
				return
			}
			list.Refresh()
		}()
	}
	c := container.NewBorder(
		container.NewBorder(
			container.NewHBox(
				widget.NewLabel("Add Recipient"),
				layout.NewSpacer(),
				widget.NewButton("Cancel", func() {
					modal.Hide()
				}),
			),
			nil,
			nil,
			nil,
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
