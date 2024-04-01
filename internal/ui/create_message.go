package ui

import (
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/model"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	widget2 "fyne.io/x/fyne/widget"
)

func (u *ui) makeCreateMessageWindow() (fyne.Window, error) {
	currentChar := *u.CurrentChar()
	w := u.app.NewWindow("New message")
	fromLabel := widget.NewLabel("From:")
	fromInput := widget.NewEntry()
	fromInput.Disable()
	fromInput.SetPlaceHolder(currentChar.Name)
	toLabel := widget.NewLabel("To:")
	toInput := widget.NewEntry()

	ee := []model.EveEntity{}
	addButton := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		label := widget.NewLabel("Search")
		entry := widget2.NewCompletionEntry([]string{})
		content := container.New(layout.NewFormLayout(), label, entry)
		entry.OnChanged = func(search string) {
			if len(search) < 2 {
				entry.HideCompletion()
				return
			}
			var err error
			ee, err := model.FetchEveEntityCharacters(search)
			if err != nil {
				entry.HideCompletion()
				return
			}
			names := []string{}
			for _, e := range ee {
				names = append(names, e.Name)
			}
			entry.SetOptions(names)
			entry.ShowCompletion()
		}
		d := dialog.NewCustomConfirm(
			"Add recipient", "Add", "Cancel", content, func(confirmed bool) {
				if confirmed {
					ss := parseNames(toInput.Text)
					ss = append(ss, entry.Text)
					slices.Sort(ss)
					s := strings.Join(ss, ", ")
					toInput.SetText(s)
				}
			},
			w,
		)
		d.Resize(fyne.Size{Width: 300, Height: 200})
		d.Show()
	})
	toInputWrap := container.NewBorder(nil, nil, nil, addButton, toInput)
	subjectLabel := widget.NewLabel("Subject:")
	subjectInput := widget.NewEntry()
	form := container.New(layout.NewFormLayout(), fromLabel, fromInput, toLabel, toInputWrap, subjectLabel, subjectInput)
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
		err := sendMail(currentChar.ID, subjectInput.Text, selected.ID, bodyInput.Text)
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

func parseNames(t string) []string {
	// split string into slice of names
	names := strings.Split(t, ",")
	for i, s := range names {
		names[i] = strings.Trim(s, " ")
	}
	// remove empty names from slice
	temp := names[:0]
	for _, s := range names {
		if len(s) == 0 {
			continue
		}
		temp = append(temp, s)
	}
	names = temp
	return names
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
