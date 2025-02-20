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

func (u *BaseUI) showSendMessageWindow(mode SendMessageMode, mail *app.CharacterMail) {
	w, err := u.makeSendMessageWindow(mode, mail)
	if err != nil {
		slog.Error("create send message window", "error", err)
	} else {
		w.Show()
	}
}

func (u *BaseUI) makeSendMessageWindow(mode SendMessageMode, mail *app.CharacterMail) (fyne.Window, error) {
	currentChar := u.CurrentCharacter()
	var title string
	if u.IsMobile() {
		title = "New message"
	} else {
		title = u.MakeWindowTitle(fmt.Sprintf("New message [%s]", currentChar.EveCharacter.Name))
	}
	w := u.FyneApp.NewWindow(title)

	toInput := widget.NewEntry()
	// toInput.MultiLine = true // FIXME: Does not work with columns layout
	// toInput.Wrapping = fyne.TextWrapWord
	toInput.Validator = newNonEmptyStringValidator()
	toInput.ActionItem = widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		u.showAddDialog(w, toInput, currentChar.ID)
	})

	fromInput := widget.NewEntry()
	fromInput.PlaceHolder = currentChar.EveCharacter.Name
	fromInput.Disable()

	subjectInput := widget.NewEntry()
	subjectInput.Validator = newNonEmptyStringValidator()

	bodyInput := widget.NewEntry()
	bodyInput.MultiLine = true
	bodyInput.SetMinRowsVisible(14)
	bodyInput.Validator = newNonEmptyStringValidator()
	bodyInput.PlaceHolder = "Compose message"

	if mail != nil {
		const sep = "\n\n--------------------------------\n"
		switch mode {
		case SendMessageReply:
			r := mailrecipient.NewFromEntities([]*app.EveEntity{mail.From})
			toInput.SetText(r.String())
			subjectInput.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			bodyInput.SetText(sep + mail.String())
		case SendMessageReplyAll:
			r := mailrecipient.NewFromEntities(mail.Recipients)
			toInput.SetText(r.String())
			subjectInput.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			bodyInput.SetText(sep + mail.String())
		case SendMessageForward:
			subjectInput.SetText(fmt.Sprintf("Fw: %s", mail.Subject))
			bodyInput.SetText(sep + mail.String())
		default:
			return nil, fmt.Errorf("undefined mode for create message: %v", mode)
		}
	}
	sendButton := widget.NewButton("Send", func() {
		ctx := context.TODO()
		showErrorDialog := func(message string) {
			d := dialog.NewInformation("Failed to send message", message, w)
			d.Show()
		}
		recipients := mailrecipient.NewFromText(toInput.Text)
		ee2, err := u.EveUniverseService.ResolveUncleanEveEntities(ctx, recipients.ToEveEntitiesUnclean())
		if err != nil {
			showErrorDialog(err.Error())
			return
		}
		_, err = u.CharacterService.SendCharacterMail(
			ctx,
			currentChar.ID,
			subjectInput.Text,
			ee2,
			bodyInput.Text,
		)
		if err != nil {
			showErrorDialog(err.Error())
			return
		}
		w.Hide()
	})
	sendButton.Importance = widget.HighImportance
	sendButton.Disable()
	checkFields := func(_ string) {
		if toInput.Validate() != nil || subjectInput.Validate() != nil || bodyInput.Validate() != nil {
			sendButton.Disable()
		} else {
			sendButton.Enable()
		}
	}
	toInput.OnChanged = checkFields
	subjectInput.OnChanged = checkFields
	bodyInput.OnChanged = checkFields

	colums := kxlayout.NewColumns(60)
	frame := container.NewBorder(
		container.NewVBox(
			container.New(colums, widget.NewLabel("From"), fromInput),
			container.New(colums, widget.NewLabel("To"), toInput),
			container.New(colums, widget.NewLabel("Subject"), subjectInput),
		),
		container.NewHBox(sendButton),
		nil,
		nil,
		bodyInput,
	)
	w.SetContent(frame)
	w.Resize(fyne.NewSize(600, 500))
	return w, nil
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
