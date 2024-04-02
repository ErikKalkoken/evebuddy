package ui

import (
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/model"
	"example/esiapp/internal/widgets"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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
	w := u.app.NewWindow("New message")
	fromLabel := widget.NewLabel("From:")
	fromInput := widget.NewEntry()
	fromInput.Disable()
	fromInput.SetPlaceHolder(currentChar.Name)
	toLabel := widget.NewLabel("To:")
	toInput := widget.NewEntry()
	subjectLabel := widget.NewLabel("Subject:")
	subjectInput := widget.NewEntry()
	bodyInput := widget.NewEntry()
	bodyInput.MultiLine = true

	if mail != nil {
		switch mode {
		case CreateMessageReply:
			r := NewRecipients()
			r.add(NewRecipientFromEntity(mail.From))
			toInput.SetText(r.String())
			subjectInput.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			bodyInput.SetText(mail.ToString(myDateTime))
		case CreateMessageReplyAll:
			r := NewRecipientsFromEntities(mail.Recipients)
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
		showAddDialog(w, toInput, currentChar.ID)
	})
	toInputWrap := container.NewBorder(nil, nil, nil, addButton, toInput)
	form := container.New(layout.NewFormLayout(), fromLabel, fromInput, toLabel, toInputWrap, subjectLabel, subjectInput)
	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		w.Hide()
	})
	sendButton := widget.NewButtonWithIcon("Send", theme.ConfirmIcon(), func() {
		recipients, err := badgeStringToEsiRecipients(toInput.Text)
		if err != nil {
			slog.Error("Failed to resolve names", "error", err)
			return
		}
		err = sendMail(currentChar.ID, subjectInput.Text, recipients, bodyInput.Text)
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
	w.Resize(fyne.NewSize(600, 500))
	return w, nil
}

func badgeStringToEsiRecipients(s string) ([]esi.MailRecipient, error) {
	r := NewRecipientsFromText(s)
	n := r.names()
	resp, err := esi.ResolveEntityNames(httpClient, n)
	var rr []esi.MailRecipient
	for _, o := range resp.Alliances {
		rr = append(rr, esi.MailRecipient{ID: o.ID, Type: "character"})
	}
	for _, o := range resp.Characters {
		rr = append(rr, esi.MailRecipient{ID: o.ID, Type: "corporation"})
	}
	for _, o := range resp.Corporations {
		rr = append(rr, esi.MailRecipient{ID: o.ID, Type: "alliance"})
	}
	return rr, err
}

func showAddDialog(w fyne.Window, toInput *widget.Entry, characterID int32) {
	label := widget.NewLabel("Search")
	entry := widgets.NewCompletionEntry([]string{})
	content := container.New(layout.NewFormLayout(), label, entry)
	entry.OnChanged = func(search string) {
		if len(search) < 3 {
			entry.HideCompletion()
			return
		}
		names, err := makeRecipientOptions(search)
		if err != nil {
			entry.HideCompletion()
			return
		}
		entry.SetOptions(names)
		entry.ShowCompletion()
		go func() {
			token, err := model.FetchToken(characterID)
			if err != nil {
				slog.Error("Failed to fetch token", "error", err)
				return
			}
			err = ensureFreshToken(token)
			if err != nil {
				slog.Error("Failed to refresh token", "error", err)
				return
			}
			categories := []esi.SearchCategory{
				esi.SearchCategoryCorporation,
				esi.SearchCategoryCharacter,
				esi.SearchCategoryAlliance,
			}
			r, err := esi.Search(httpClient, characterID, search, categories, token.AccessToken)
			if err != nil {
				slog.Error("Failed to search on ESI", "error", err)
				return
			}
			ids := slices.Concat(r.Alliance, r.Character, r.Corporation)
			err = addMissingEveEntities(ids)
			if err != nil {
				slog.Error("Failed to fetch missing IDs", "error", err)
				return
			}
			names, err := makeRecipientOptions(search)
			if err != nil {
				slog.Error("Failed to make name options", "error", err)
				return
			}
			entry.SetOptions(names)
			entry.ShowCompletion()
		}()
	}
	d := dialog.NewCustomConfirm(
		"Add recipient", "Add", "Cancel", content, func(confirmed bool) {
			if confirmed {
				r := NewRecipientsFromText(toInput.Text)
				r.add(NewRecipientFromText(entry.Text))
				toInput.SetText(r.String())
			}
		},
		w,
	)
	d.Resize(fyne.Size{Width: 500, Height: 175})
	d.Show()
}

func makeRecipientOptions(search string) ([]string, error) {
	ee, err := model.FetchEveEntityNameSearch(search)
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, e := range ee {
		r := NewRecipientFromEntity(e)
		names = append(names, r.String())
	}
	return names, nil
}

func sendMail(characterID int32, subject string, recipients []esi.MailRecipient, body string) error {
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
		Recipients: recipients,
	}
	_, err = esi.SendMail(httpClient, characterID, token.AccessToken, m)
	if err != nil {
		return err
	}
	return nil
}
