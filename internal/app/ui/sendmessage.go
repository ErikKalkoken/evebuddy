package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/mailrecipient"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
)

const (
	createMessageNew = iota
	createMessageReply
	createMessageReplyAll
	createMessageForward
)

func (u *UI) showSendMessageWindow(mode int, mail *app.CharacterMail) {
	w, err := u.makeSendMessageWindow(mode, mail)
	if err != nil {
		slog.Error("create send message window", "error", err)
	} else {
		w.Show()
	}
}

func (u *UI) makeSendMessageWindow(mode int, mail *app.CharacterMail) (fyne.Window, error) {
	currentChar := *u.currentCharacter()
	w := u.fyneApp.NewWindow(u.makeWindowTitle(fmt.Sprintf("New message [%s]", currentChar.EveCharacter.Name)))

	fromInput := widget.NewEntry()
	fromInput.Disable()
	fromInput.SetPlaceHolder(currentChar.EveCharacter.Name)

	toInput := widget.NewEntry()
	toInput.MultiLine = true
	toInput.Wrapping = fyne.TextWrapWord
	toInput.Validator = newNonEmptyStringValidator()

	subjectInput := widget.NewEntry()
	subjectInput.Validator = newNonEmptyStringValidator()

	bodyInput := widget.NewEntry()
	bodyInput.MultiLine = true
	bodyInput.SetMinRowsVisible(14)
	bodyInput.Validator = newNonEmptyStringValidator()

	if mail != nil {
		const sep = "\n\n--------------------------------\n"
		switch mode {
		case createMessageReply:
			r := mailrecipient.NewFromEntities([]*app.EveEntity{mail.From})
			toInput.SetText(r.String())
			subjectInput.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			bodyInput.SetText(sep + mail.String())
		case createMessageReplyAll:
			r := mailrecipient.NewFromEntities(mail.Recipients)
			toInput.SetText(r.String())
			subjectInput.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			bodyInput.SetText(sep + mail.String())
		case createMessageForward:
			subjectInput.SetText(fmt.Sprintf("Fw: %s", mail.Subject))
			bodyInput.SetText(sep + mail.String())
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
				Text:   "",
				Widget: bodyInput,
			},
		},
		OnSubmit: func() {
			recipients := mailrecipient.NewFromText(toInput.Text)
			err := checkInput(subjectInput.Text, recipients, bodyInput.Text)
			if err == nil {
				err = func() error {
					ctx := context.TODO()
					eeUnclean := recipients.ToEveEntitiesUnclean()
					ee2, err := u.EveUniverseService.ResolveUncleanEveEntities(ctx, eeUnclean)
					if err != nil {
						return err
					}
					_, err = u.CharacterService.SendCharacterMail(ctx, currentChar.ID, subjectInput.Text, ee2, bodyInput.Text)
					if err != nil {
						return err
					}
					names := make([]string, len(ee2))
					for i, e := range ee2 {
						names[i] = e.Name
					}
					slices.Sort(names)
					s := strings.Join(names, ", ")
					u.statusBarArea.SetInfo(fmt.Sprintf("Message sent to %s", s))
					return nil
				}()
			}
			if err != nil {
				t := "Failed to send mail"
				slog.Error(t, "err", err)
				d := newErrorDialog(t, err, u.window)
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

func checkInput(subject string, r *mailrecipient.MailRecipients, body string) error {
	if len(subject) == 0 {
		return errors.New("missing subject")
	}
	if len(body) == 0 {
		return errors.New("missing text")
	}
	if r.Size() == 0 {
		return errors.New("missing recipients")
	}
	return nil
}

func (u *UI) showAddDialog(w fyne.Window, toInput *widget.Entry, characterID int32) {
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
			missingIDs, err := u.CharacterService.AddEveEntitiesFromCharacterSearchESI(context.TODO(), characterID, search)
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
				r := mailrecipient.NewFromText(toInput.Text)
				r.AddFromText(entry.Text)
				toInput.SetText(r.String())
			}
		},
		w,
	)
	d.Resize(fyne.Size{Width: 500, Height: 175})
	d.Show()
}

func (u *UI) makeRecipientOptions(search string) ([]string, error) {
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
