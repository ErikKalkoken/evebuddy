package ui

import (
	"example/esiapp/internal/core"
	"example/esiapp/internal/storage"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (e *esiApp) newCharacters(charID int32) *fyne.Container {
	shareItem := makeShareItem()
	buttonAdd := newContextMenuButton(
		"Manage Characters", fyne.NewMenu("",
			fyne.NewMenuItem("Add Character", func() {
				info := dialog.NewInformation(
					"Add Character",
					"Please follow instructions in your browser to add a new character.",
					e.main,
				)
				info.Show()
				_, err := core.AddCharacter()
				if err != nil {
					log.Printf("Failed to add a new character: %v", err)
				}
			}),
			shareItem,
		))

	currentUser := container.NewHBox()
	c, err := storage.FetchCharacter(charID)
	if err != nil {
		currentUser.Add(widget.NewLabel("No characters"))
		log.Print("No token found")
	} else {
		image := canvas.NewImageFromURI(c.PortraitURL(64))
		image.FillMode = canvas.ImageFillOriginal
		currentUser.Add(image)
		currentUser.Add(widget.NewLabel(c.Name))
	}
	currentUser.Add(layout.NewSpacer())
	currentUser.Add(buttonAdd)
	return currentUser
}

func makeShareItem() *fyne.MenuItem {
	cc, err := storage.FetchAllCharacters()
	if err != nil {
		log.Fatal(err)
	}
	shareItem := fyne.NewMenuItem("Switch character", nil)

	var items []*fyne.MenuItem
	for _, c := range cc {
		item := fyne.NewMenuItem(c.Name, func() { log.Printf("selected %v", c.Name) })
		items = append(items, item)
	}
	shareItem.ChildMenu = fyne.NewMenu("", items...)
	return shareItem
}
