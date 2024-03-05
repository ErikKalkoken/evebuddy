package main

import (
	"fmt"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"example/esiapp/internal/core"
	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"

	"github.com/microcosm-cc/bluemonday"
)

const (
	myDateTime = "2006-01-02 15:04"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

	if err := storage.Initialize(); err != nil {
		log.Fatal(err)
	}

	myApp := app.New()
	myWindow := myApp.NewWindow("Eve Online App")

	buttonAdd := widget.NewButton("Add Character", func() {
		scopes := []string{
			"esi-characters.read_contacts.v1",
			"esi-universe.read_structures.v1",
			"esi-mail.read_mail.v1",
		}
		ssoToken, err := sso.Authenticate(scopes)
		if err != nil {
			log.Fatal(err)
		}
		character := storage.Character{
			ID:   ssoToken.CharacterID,
			Name: ssoToken.CharacterName,
		}
		if err = character.Save(); err != nil {
			log.Fatal(err)
		}
		token := storage.Token{
			AccessToken:  ssoToken.AccessToken,
			Character:    character,
			ExpiresAt:    ssoToken.ExpiresAt,
			RefreshToken: ssoToken.RefreshToken,
			TokenType:    ssoToken.TokenType,
		}
		if err = token.Save(); err != nil {
			log.Fatal(err)
		}
		info := dialog.NewInformation(
			"Authentication completed",
			fmt.Sprintf("Authenticated: %v", ssoToken.CharacterName),
			myWindow,
		)
		info.Show()
	})

	currentUser := container.NewHBox()
	token, err := storage.FirstToken()
	if err != nil {
		currentUser.Add(widget.NewLabel("Not authenticated"))
		log.Print("No token found")
	} else {
		image := canvas.NewImageFromURI(token.IconUrl(64))
		image.FillMode = canvas.ImageFillOriginal
		currentUser.Add(image)
		currentUser.Add(widget.NewLabel(token.Character.Name))
	}
	currentUser.Add(buttonAdd)

	buttonFetch := widget.NewButton("Fetch mail", func() {
		err := core.UpdateMails(93330670)
		if err != nil {
			log.Fatal(err)
		}
	})

	mails, err := storage.FetchMail(93330670)
	if err != nil {
		log.Fatalf("Failed to fetch mail: %v", err)
	}
	mailHeaderSubject := widget.NewLabel("")
	mailHeaderSubject.TextStyle = fyne.TextStyle{Bold: true}
	mailHeaderSubject.Truncation = fyne.TextTruncateEllipsis

	mailHeaderFrom := widget.NewLabel("")
	mailHeaderTimestamp := widget.NewLabel("")
	mailHeader := container.NewVBox(mailHeaderSubject, mailHeaderFrom, mailHeaderTimestamp)
	mailBody := widget.NewLabel("Text")
	mailBody.Wrapping = fyne.TextWrapBreak

	detail := container.NewBorder(mailHeader, nil, nil, nil, container.NewVScroll(mailBody))

	blue := bluemonday.StrictPolicy()
	headers := widget.NewList(
		func() int {
			return len(mails)
		},
		func() fyne.CanvasObject {
			from := widget.NewLabel("from")
			timestamp := widget.NewLabel("timestamp")
			subject := widget.NewLabel("subject")
			subject.TextStyle = fyne.TextStyle{Bold: true}
			subject.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(
				container.NewHBox(from, layout.NewSpacer(), timestamp), subject,
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			mail := mails[i]
			// style := fyne.TextStyle{Bold: !mail.IsRead}
			parent := o.(*fyne.Container)
			top := parent.Objects[0].(*fyne.Container)
			from := top.Objects[0].(*widget.Label)
			from.SetText(mail.From.Name)
			timestamp := top.Objects[2].(*widget.Label)
			timestamp.SetText(mail.TimeStamp.Format(myDateTime))
			subject := parent.Objects[1].(*widget.Label)
			subject.SetText(mail.Subject)
		})
	headers.OnSelected = func(id widget.ListItemID) {
		mail := mails[id]
		mailHeaderSubject.SetText(mail.Subject)
		mailHeaderFrom.SetText("From: " + mail.From.Name)
		mailHeaderTimestamp.SetText("Received: " + mail.TimeStamp.Format(myDateTime))
		text := strings.ReplaceAll(mail.Body, "<br>", "\n")
		mailBody.SetText(blue.Sanitize(text))
	}

	main := container.NewHSplit(headers, detail)

	content := container.NewBorder(currentUser, buttonFetch, nil, nil, main)
	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.ShowAndRun()
}
