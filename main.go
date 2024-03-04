package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"example/esiapp/internal/esi"
	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"
)

func main() {
	db := storage.Open()

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
		if err = db.Save(&character).Error; err != nil {
			log.Fatal(err)
		}
		token := storage.Token{
			AccessToken:  ssoToken.AccessToken,
			Character:    character,
			ExpiresAt:    ssoToken.ExpiresAt,
			RefreshToken: ssoToken.RefreshToken,
			TokenType:    ssoToken.TokenType,
		}
		if err = db.Where("character_id = ?", character.ID).Save(&token).Error; err != nil {
			log.Fatal(err)
		}
		info := dialog.NewInformation("Authentication completed", fmt.Sprintf("Authenticated: %v", ssoToken.CharacterName), myWindow)
		info.Show()
	})

	currentUser := container.NewHBox()
	var token storage.Token
	if db.Joins("Character").First(&token).Error != nil {
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
		mail, err := esi.FetchMailHeaders(token.CharacterID, token.AccessToken)
		if err != nil {
			log.Println(err)
		} else {
			fmt.Println(mail)
		}
	})

	middle := widget.NewLabel("PLACEHOLDER")
	content := container.NewBorder(currentUser, nil, nil, buttonFetch, middle)
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
