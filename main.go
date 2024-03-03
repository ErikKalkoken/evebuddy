package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"
)

func main() {
	conn := storage.Open()
	defer conn.Close()

	myApp := app.New()
	myWindow := myApp.NewWindow("Eve Online App")

	middle := widget.NewLabel("content")

	buttonAdd := widget.NewButton("Add Character", func() {
		scopes := []string{"esi-characters.read_contacts.v1", "esi-universe.read_structures.v1"}
		token, err := sso.Authenticate(scopes)
		if err != nil {
			log.Fatal(err)
		}
		token.Store()
		middle.SetText(fmt.Sprintf("Authenticated: %v", token.CharacterName))
	})

	buttonLoad := widget.NewButton("Load Character", func() {
		token, err := storage.FindToken(93330670)
		if err != nil {
			log.Fatal(err)
		}
		middle.SetText(fmt.Sprintf("Found token: %v", token.CharacterName))
	})

	content := container.NewBorder(buttonAdd, buttonLoad, nil, nil, middle)
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
