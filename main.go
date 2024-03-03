package main

import (
	"example/esiapp/internal/sso"
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Eve Online App")

	middle := widget.NewLabel("content")

	button := widget.NewButton("click me", func() {
		scopes := []string{"esi-characters.read_contacts.v1", "esi-universe.read_structures.v1"}
		token, err := sso.Authenticate(scopes)
		if err != nil {
			log.Fatal(err)
		}
		middle.SetText(fmt.Sprintf("Authenticated: %v", token.CharacterName))
		middle.Refresh()
	})

	content := container.NewBorder(button, nil, nil, nil, middle)
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
