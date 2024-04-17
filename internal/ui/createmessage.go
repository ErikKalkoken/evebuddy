package ui

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"example/evebuddy/internal/model"
	"example/evebuddy/internal/widgets"
)

const (
	CreateMessageNew = iota
	CreateMessageReply
	CreateMessageReplyAll
	CreateMessageForward
)

func (u *ui) ShowCreateMessageWindow(mode int, mail *model.Mail) {
	w, err := u.makeCreateMessageWindow(mode, mail)
	if err != nil {
		slog.Error("failed to create new message window", "error", err)
	} else {
		w.Show()
	}
}

func (u *ui) makeCreateMessageWindow(mode int, mail *model.Mail) (fyne.Window, error) {
	currentChar := *u.CurrentChar()
	w := u.app.NewWindow(fmt.Sprintf("New message [%s]", currentChar.Name))
	fromLabel := widget.NewLabel("From:")
	fromInput := widget.NewEntry()
	fromInput.Disable()
	fromInput.SetPlaceHolder(currentChar.Name)
	toLabel := widget.NewLabel("To:")
	toInput := widget.NewEntry()
	toInput.MultiLine = true
	toInput.Wrapping = fyne.TextWrapWord
	subjectLabel := widget.NewLabel("Subject:")
	subjectInput := widget.NewEntry()
	bodyInput := widget.NewEntry()
	bodyInput.MultiLine = true

	if mail != nil {
		switch mode {
		case CreateMessageReply:
			r := NewMailRecipientsFromEntities([]model.EveEntity{mail.From})
			toInput.SetText(r.String())
			subjectInput.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			bodyInput.SetText(mail.ToString(myDateTime))
		case CreateMessageReplyAll:
			r := NewMailRecipientsFromEntities(mail.Recipients)
			toInput.SetText(r.String())
			subjectInput.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			bodyInput.SetText(mail.ToString(myDateTime))
		case CreateMessageForward:
			subjectInput.SetText(fmt.Sprintf("Fw: %s", mail.Subject))
			bodyInput.SetText(mail.ToString(myDateTime))
		default:
			return nil, fmt.Errorf("undefined mode for create message: %v", mode)
		}
	}

	addButton := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		u.showAddDialog(w, toInput, currentChar.ID)
	})
	toInputWrap := container.NewBorder(nil, nil, nil, addButton, toInput)
	form := container.New(layout.NewFormLayout(), fromLabel, fromInput, toLabel, toInputWrap, subjectLabel, subjectInput)
	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		w.Hide()
	})
	sendButton := widget.NewButtonWithIcon("Send", theme.ConfirmIcon(), func() {
		err := func() error {
			recipients := NewMailRecipientsFromText(toInput.Text)
			eeUnclean := recipients.ToEveEntitiesUnclean()
			ee2, err := u.service.ResolveUncleanEveEntities(eeUnclean)
			if err != nil {
				return err
			}
			if err := u.service.SendMail(currentChar.ID, subjectInput.Text, ee2, bodyInput.Text); err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			// TODO: Replace with user friendly error messages
			slog.Error(err.Error())
			d := dialog.NewError(err, w)
			d.Show()
			return
		}
		w.Hide()
	})
	sendButton.Importance = widget.HighImportance
	buttons := container.NewHBox(
		cancelButton,
		layout.NewSpacer(),
		sendButton,
	)
	content := container.NewBorder(form, buttons, nil, nil, bodyInput)
	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 500))
	return w, nil
}

func (u *ui) showAddDialog(w fyne.Window, toInput *widget.Entry, characterID int32) {
	label := widget.NewLabel("Search")
	entry := widgets.NewCompletionEntry([]string{})
	content := container.New(layout.NewFormLayout(), label, entry)
	entry.OnChanged = func(search string) {
		if len(search) < 3 {
			entry.HideCompletion()
			return
		}
		names, err := u.makeRecipientOptions(search)
		if err != nil {
			entry.HideCompletion()
			return
		}
		entry.SetOptions(names)
		entry.ShowCompletion()
		go func() {
			missingIDs, err := u.service.AddEveEntitiesFromESISearch(characterID, search)
			if err != nil {
				slog.Error("Failed to search names", "search", "search", "error", err)
				return
			}
			if len(missingIDs) == 0 {
				return // no need to update when not changed
			}
			names2, err := u.makeRecipientOptions(search)
			if err != nil {
				slog.Error("Failed to make name options", "error", err)
				return
			}
			entry.SetOptions(names2)
			entry.ShowCompletion()
		}()
	}
	d := dialog.NewCustomConfirm(
		"Add recipient", "Add", "Cancel", content, func(confirmed bool) {
			if confirmed {
				r := NewMailRecipientsFromText(toInput.Text)
				r.AddFromText(entry.Text)
				toInput.SetText(r.String())
			}
		},
		w,
	)
	d.Resize(fyne.Size{Width: 500, Height: 175})
	d.Show()
}

func (u *ui) makeRecipientOptions(search string) ([]string, error) {
	ee, err := u.service.ListEveEntitiesByPartialName(search)
	if err != nil {
		return nil, err
	}
	rr := NewMailRecipientsFromEntities(ee)
	oo := rr.ToOptions()
	return oo, nil
}
