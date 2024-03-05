package main

import (
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"example/esiapp/internal/core"
	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"
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
		info := dialog.NewInformation("Authentication completed", fmt.Sprintf("Authenticated: %v", ssoToken.CharacterName), myWindow)
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
		err := core.FetchMail(93330670)
		if err != nil {
			log.Fatal(err)
		}
	})

	mails, err := storage.FetchMail(93330670)
	if err != nil {
		log.Fatalf("Failed to fetch mail: %v", err)
	}

	list := widget.NewList(
		func() int {
			return len(mails)
		},
		func() fyne.CanvasObject {
			c := theme.ForegroundColor()
			from := canvas.NewText("from", c)
			timestamp := canvas.NewText("timestamp", c)
			subject := canvas.NewText("subject", c)
			subject.TextSize = theme.TextSize() * 1.15
			return container.NewVBox(
				container.NewHBox(from, layout.NewSpacer(), timestamp), subject,
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			mail := mails[i]
			style := fyne.TextStyle{Bold: !mail.IsRead}
			parent := o.(*fyne.Container)
			top := parent.Objects[0].(*fyne.Container)
			from := top.Objects[0].(*canvas.Text)
			from.Text = mail.From.Name
			from.TextStyle = style
			timestamp := top.Objects[2].(*canvas.Text)
			timestamp.Text = mail.TimeStamp.Format(time.DateTime)
			timestamp.TextStyle = style
			subject := parent.Objects[1].(*canvas.Text)
			subject.Text = mail.Subject
			subject.TextStyle = style
			from.Refresh()
			timestamp.Refresh()
			subject.Refresh()
		})

	content := container.NewBorder(currentUser, buttonFetch, nil, nil, list)
	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.ShowAndRun()

	// scopes := []string{"esi-characters.read_contacts.v1", "esi-universe.read_structures.v1"}
	// token, err := sso.Authenticate(scopes)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// contacts := esi.FetchContacts(token.CharacterID, token.AccessToken)
	// fmt.Println(contacts)
}
