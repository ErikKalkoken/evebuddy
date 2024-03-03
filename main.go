package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"
)

func main() {
	db := storage.Open()

	myApp := app.New()
	myWindow := myApp.NewWindow("Eve Online App")

	middle := widget.NewLabel("content")

	buttonAdd := widget.NewButton("Add Character", func() {
		scopes := []string{"esi-characters.read_contacts.v1", "esi-universe.read_structures.v1"}
		token, err := sso.Authenticate(scopes)
		if err != nil {
			log.Fatal(err)
		}
		if err = db.Save(token).Error; err != nil {
			log.Fatal(err)
		}
		middle.SetText(fmt.Sprintf("Authenticated: %v", token.CharacterName))
	})

	var tokens []storage.Token
	result := db.Find(&tokens)
	if result.Error != nil {
		log.Fatal(result.Error)
	}

	characters := container.NewVBox(buttonAdd)
	for _, t := range tokens {
		image := canvas.NewImageFromURI(t.IconUrl(128))
		image.FillMode = canvas.ImageFillOriginal
		characters.Add(image)
	}

	content := container.NewBorder(nil, nil, characters, nil, middle)
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
