package ui

import (
	"errors"
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

	fromInput := widget.NewEntry()
	fromInput.Disable()
	fromInput.SetPlaceHolder(currentChar.Name)

	toInput := widget.NewEntry()
	toInput.MultiLine = true
	toInput.Wrapping = fyne.TextWrapWord
	toInput.Validator = NewNonEmptyStringValidator()

	subjectInput := widget.NewEntry()
	subjectInput.Validator = NewNonEmptyStringValidator()

	bodyInput := widget.NewEntry()
	bodyInput.MultiLine = true
	bodyInput.SetMinRowsVisible(14)
	bodyInput.Validator = NewNonEmptyStringValidator()

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
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "From", Widget: fromInput},
			{
				Text:   "To",
				Widget: toInputWrap,
			},
			{
				Text:   "Subject",
				Widget: subjectInput,
			},
			{
				Text:   "Text",
				Widget: bodyInput,
			},
		},
		OnSubmit: func() {
			recipients := NewMailRecipientsFromText(toInput.Text)
			err := checkInput(subjectInput.Text, recipients, bodyInput.Text)
			if err == nil {
				err = func() error {
					eeUnclean := recipients.ToEveEntitiesUnclean()
					ee2, err := u.service.ResolveUncleanEveEntities(eeUnclean)
					if err != nil {
						return err
					}
					_, err = u.service.SendMail(currentChar.ID, subjectInput.Text, ee2, bodyInput.Text)
					if err != nil {
						return err
					}
					return nil
				}()
			}
			if err != nil {
				slog.Error(err.Error())
				d := dialog.NewInformation("Failed to send mail", fmt.Sprintf("An error occurred: %s", err), w)
				d.Show()
				return
			}
			w.Hide()
		},
		OnCancel: func() {
			w.Hide()
		},
	}

	w.SetContent(form)
	w.Resize(fyne.NewSize(600, 500))
	return w, nil
}

func checkInput(subject string, recipients *MailRecipients, body string) error {
	if len(subject) == 0 {
		return errors.New("missing subject")
	}
	if len(body) == 0 {
		return errors.New("missing text")
	}
	if recipients.Size() == 0 {
		return errors.New("missing recipients")
	}
	return nil
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

func NewNonEmptyStringValidator() fyne.StringValidator {
	myErr := errors.New("can not be empty")
	return func(text string) error {
		if len(text) == 0 {
			return myErr
		}
		return nil
	}
}
