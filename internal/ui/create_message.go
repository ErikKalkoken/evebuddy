package ui

import (
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/model"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	widget2 "fyne.io/x/fyne/widget"
)

func (u *ui) makeCreateMessageWindow() (fyne.Window, error) {
	character := *u.CurrentChar()
	w := u.app.NewWindow(fmt.Sprintf("New message [%s]", character.Name))
	toLabel := widget.NewLabel("To:")
	toInput := widget2.NewCompletionEntry([]string{})
	ee := []model.EveEntity{}
	toInput.OnChanged = func(s string) {
		if len(s) < 3 {
			toInput.HideCompletion()
			return
		}
		var err error
		ee, err = model.FetchEveEntityCharacters(s)
		if err != nil {
			slog.Error("Failed to fetch character names", "error", "err")
			toInput.HideCompletion()
			return
		}
		names := []string{}
		for _, e := range ee {
			names = append(names, e.Name)
		}
		toInput.SetOptions(names)
		toInput.ShowCompletion()
	}
	subjectLabel := widget.NewLabel("Subject:")
	subjectInput := widget.NewEntry()
	form := container.New(layout.NewFormLayout(), toLabel, toInput, subjectLabel, subjectInput)
	bodyInput := widget.NewEntry()
	bodyInput.MultiLine = true
	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		w.Hide()
	})
	sendButton := widget.NewButtonWithIcon("Send", theme.ConfirmIcon(), func() {
		name := toInput.Text
		var selected *model.EveEntity
		for _, e := range ee {
			if e.Name == name {
				selected = &e
				break
			}
		}
		if selected == nil {
			slog.Error("Failed to match recipient", "name", name)
			return
		}
		err := sendMail(character.ID, subjectInput.Text, selected.ID, bodyInput.Text)
		if err != nil {
			slog.Error("Failed to send mail", "error", err)
		} else {
			w.Hide()
		}
	})
	sendButton.Importance = widget.HighImportance
	buttons := container.NewHBox(
		cancelButton,
		layout.NewSpacer(),
		sendButton,
	)
	content := container.NewBorder(form, buttons, nil, nil, bodyInput)
	w.SetContent(content)
	w.Resize(fyne.NewSize(400, 300))
	return w, nil
}

func sendMail(characterID int32, subject string, recipientID int32, body string) error {
	token, err := model.FetchToken(characterID)
	if err != nil {
		return err
	}
	err = ensureFreshToken(token)
	if err != nil {
		return err
	}
	m := esi.MailSend{
		Body:       body,
		Subject:    subject,
		Recipients: []esi.MailRecipient{{ID: recipientID, Type: "character"}},
	}
	_, err = esi.SendMail(httpClient, characterID, token.AccessToken, m)
	if err != nil {
		return err
	}
	return nil
}
