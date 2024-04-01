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
)

func (u *ui) makeCreateMessageWindow() fyne.Window {
	character := *u.CurrentChar()
	w := u.app.NewWindow(fmt.Sprintf("New message [%s]", character.Name))
	toLabel := widget.NewLabel("To:")
	toInput := widget.NewEntry()
	toInput.PlaceHolder = "Yuna Kobayashi"
	subjectLabel := widget.NewLabel("Subject:")
	subjectInput := widget.NewEntry()
	form := container.New(layout.NewFormLayout(), toLabel, toInput, subjectLabel, subjectInput)
	bodyInput := widget.NewEntry()
	bodyInput.MultiLine = true
	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		w.Hide()
	})
	sendButton := widget.NewButtonWithIcon("Send", theme.ConfirmIcon(), func() {
		err := sendMail(character.ID, subjectInput.Text, bodyInput.Text)
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
	return w
}

func sendMail(characterID int32, subject string, body string) error {
	token, err := model.FetchToken(characterID)
	if err != nil {
		return err
	}
	err = ensureFreshToken(token)
	if err != nil {
		return err
	}
	m := esi.MailSend{Body: body, Subject: subject, Recipients: []esi.MailRecipient{{ID: 95391458, Type: "character"}}}
	_, err = esi.SendMail(httpClient, characterID, token.AccessToken, m)
	if err != nil {
		return err
	}
	return nil
}
