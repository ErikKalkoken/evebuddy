package ui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/mailrecipient"
)

func (u *BaseUI) showSendMessageWindow(character *app.Character, mode SendMessageMode, mail *app.CharacterMail) {
	var title string
	if u.IsMobile() {
		title = "New message"
	} else {
		title = u.MakeWindowTitle(fmt.Sprintf("New message [%s]", character.EveCharacter.Name))
	}
	w := u.FyneApp.NewWindow(title)
	page := u.makeSendMessagePage(character, mode, mail, w)
	w.SetContent(page)
	w.Resize(fyne.NewSize(600, 500))
	w.Show()
}

func (u *BaseUI) makeSendMessagePage(character *app.Character, mode SendMessageMode, mail *app.CharacterMail, w fyne.Window) fyne.CanvasObject {
	to := widget.NewEntry()
	// toInput.MultiLine = true // FIXME: Does not work with columns layout
	// toInput.Wrapping = fyne.TextWrapWord
	to.Validator = newNonEmptyStringValidator()
	to.ActionItem = widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		u.showAddDialog(w, to, character.ID)
	})

	from := widget.NewEntry()
	from.PlaceHolder = character.EveCharacter.Name
	from.Disable()

	subject := widget.NewEntry()
	subject.Validator = newNonEmptyStringValidator()

	body := widget.NewEntry()
	body.MultiLine = true
	body.SetMinRowsVisible(14)
	body.Validator = newNonEmptyStringValidator()
	body.PlaceHolder = "Compose message"

	if mail != nil {
		const sep = "\n\n--------------------------------\n"
		switch mode {
		case SendMessageReply:
			r := mailrecipient.NewFromEntities([]*app.EveEntity{mail.From})
			to.SetText(r.String())
			subject.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			body.SetText(sep + mail.String())
		case SendMessageReplyAll:
			r := mailrecipient.NewFromEntities(mail.Recipients)
			to.SetText(r.String())
			subject.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			body.SetText(sep + mail.String())
		case SendMessageForward:
			subject.SetText(fmt.Sprintf("Fw: %s", mail.Subject))
			body.SetText(sep + mail.String())
		default:
			panic(fmt.Errorf("undefined mode for create message: %v", mode))
		}
	}
	send := widget.NewButton("Send", func() {
		ctx := context.TODO()
		showErrorDialog := func(message string) {
			d := dialog.NewInformation("Failed to send message", message, w)
			d.Show()
		}
		recipients := mailrecipient.NewFromText(to.Text)
		ee2, err := u.EveUniverseService.ResolveUncleanEveEntities(ctx, recipients.ToEveEntitiesUnclean())
		if err != nil {
			showErrorDialog(err.Error())
			return
		}
		_, err = u.CharacterService.SendCharacterMail(
			ctx,
			character.ID,
			subject.Text,
			ee2,
			body.Text,
		)
		if err != nil {
			showErrorDialog(err.Error())
			return
		}
		w.Hide()
	})
	send.Importance = widget.HighImportance
	send.Disable()
	checkFields := func(_ string) {
		if to.Validate() != nil || subject.Validate() != nil || body.Validate() != nil {
			send.Disable()
		} else {
			send.Enable()
		}
	}
	to.OnChanged = checkFields
	subject.OnChanged = checkFields
	body.OnChanged = checkFields

	colums := kxlayout.NewColumns(60)
	page := container.NewBorder(
		container.NewVBox(
			container.New(colums, widget.NewLabel("From"), from),
			container.New(colums, widget.NewLabel("To"), to),
			container.New(colums, widget.NewLabel("Subject"), subject),
		),
		container.NewHBox(send),
		nil,
		nil,
		body,
	)
	return page
}

func (u *BaseUI) showAddDialog(w fyne.Window, toInput *widget.Entry, characterID int32) {
	var dlg dialog.Dialog
	names := make([]string, 0)
	list := widget.NewList(
		func() int {
			return len(names)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("") // TODO: Show names and category in different columns, maybe also show icons
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(names) {
				return
			}
			co.(*widget.Label).SetText(names[id])
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(names) {
			list.UnselectAll()
			return
		}
		r := mailrecipient.NewFromText(toInput.Text)
		r.AddFromText(names[id])
		toInput.SetText(r.String())
		dlg.Hide()
	}
	entry := widget.NewEntry()
	entry.PlaceHolder = "Type to start searching..."
	entry.ActionItem = widget.NewIcon(theme.SearchIcon())
	entry.OnChanged = func(search string) {
		if len(search) < 3 {
			names = []string{}
			list.Refresh()
			return
		}
		var err error
		names, err = u.makeRecipientOptions(search)
		if err != nil {
			slog.Error("Failed to find names", "search", search, "error", err)
			// TODO: show error message
			return
		}
		list.Refresh()
		go func() {
			missingIDs, err := u.CharacterService.AddEveEntitiesFromCharacterSearchESI(
				context.Background(),
				characterID,
				search,
			)
			if err != nil {
				slog.Error("Failed to search names", "search", search, "error", err)
				// TODO: show error message
				return
			}
			if len(missingIDs) == 0 {
				return // no need to update when not changed
			}
			names, err = u.makeRecipientOptions(search)
			if err != nil {
				slog.Error("Failed to make name options", "error", err)
				// TODO: show error message
				return
			}
			list.Refresh()
		}()
	}
	rect := canvas.NewRectangle(color.Transparent)
	rect.StrokeColor = theme.Color(theme.ColorNameMenuBackground)
	rect.StrokeWidth = 1
	c := container.NewBorder(
		entry,
		nil,
		nil,
		nil,
		container.NewStack(rect, list),
	)
	dlg = dialog.NewCustom("Add recipient", "Cancel", c, w)
	s := w.Canvas().Size()
	var width float32
	if u.IsMobile() {
		width = s.Width
	} else {
		width = s.Width * 0.8
	}
	dlg.Resize(fyne.NewSize(width, s.Height*0.8))
	dlg.Show()
}

func (u *BaseUI) makeRecipientOptions(search string) ([]string, error) {
	ee, err := u.EveUniverseService.ListEveEntitiesByPartialName(context.TODO(), search)
	if err != nil {
		return nil, err
	}
	rr := mailrecipient.NewFromEntities(ee)
	oo := rr.ToOptions()
	return oo, nil
}

func newNonEmptyStringValidator() fyne.StringValidator {
	myErr := errors.New("can not be empty")
	return func(text string) error {
		if len(text) == 0 {
			return myErr
		}
		return nil
	}
}
