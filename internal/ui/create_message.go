package ui

import (
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/model"
	"example/esiapp/internal/widgets"
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
			toInput.SetText(mail.From.Name)
			subjectInput.SetText(fmt.Sprintf("Re: %s", mail.Subject))
			bodyInput.SetText(mail.ToString(myDateTime))
		case CreateMessageReplyAll:
			var names []string
			for _, r := range mail.Recipients {
				if r.Name == currentChar.Name {
					continue
				}
				names = append(names, r.Name)
			}
			names = append(names, mail.From.Name)
			s := makeRecipientText(names)
			toInput.SetText(s)
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
		label := widget.NewLabel("Search")
		entry := widgets.NewCompletionEntry([]string{})
		content := container.New(layout.NewFormLayout(), label, entry)
		entry.OnChanged = func(search string) {
			if len(search) < 2 {
				entry.HideCompletion()
				return
			}
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
					s := makeRecipientText(ss)
					toInput.SetText(s)
				}
			},
			w,
		)
		d.Resize(fyne.Size{Width: 500, Height: 500})
		d.Show()
	})
	toInputWrap := container.NewBorder(nil, nil, nil, addButton, toInput)
	form := container.New(layout.NewFormLayout(), fromLabel, fromInput, toLabel, toInputWrap, subjectLabel, subjectInput)
	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		w.Hide()
	})
	sendButton := widget.NewButtonWithIcon("Send", theme.ConfirmIcon(), func() {
		names := parseNames(toInput.Text)
		recipients, err := esi.ResolveEntityNames(httpClient, names)
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

func makeRecipientText(ss []string) string {
	slices.Sort(ss)
	s := strings.Join(ss, ", ")
	return s
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

func sendMail(characterID int32, subject string, recipients *esi.UniverseIDsResponse, body string) error {
	token, err := model.FetchToken(characterID)
	if err != nil {
		return err
	}
	err = ensureFreshToken(token)
	if err != nil {
		return err
	}
	rr := []esi.MailRecipient{}
	for _, r := range recipients.Characters {
		rr = append(rr, esi.MailRecipient{ID: r.ID, Type: "character"})
	}
	m := esi.MailSend{
		Body:       body,
		Subject:    subject,
		Recipients: rr,
	}
	_, err = esi.SendMail(httpClient, characterID, token.AccessToken, m)
	if err != nil {
		return err
	}
	return nil
}
