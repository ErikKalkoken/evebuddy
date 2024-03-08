package main

import (
	"example/esiapp/internal/core"
	"example/esiapp/internal/storage"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type labelItem struct {
	id   int32
	name string
}

type folders struct {
	container fyne.CanvasObject
	data      binding.ExternalUntypedList
}

func (f *folders) update(characterID int32) {
	labels, err := storage.FetchAllMailLabels(characterID)
	if err != nil {
		log.Fatal(err)
	}
	for _, l := range labels {
		f.data.Append(labelItem{id: l.ID, name: l.Name})
	}

}

func (e *esiapp) newFolders() *folders {
	list, data := makeFolderList()
	button := makeRefreshButton(e.main, e.characterID)
	c := container.NewBorder(button, nil, nil, nil, list)
	f := folders{container: c, data: data}
	return &f
}

func makeFolderList() (*widget.List, binding.ExternalUntypedList) {
	var ii []interface{}
	ii = append(ii, labelItem{id: 0, name: "All Mails"})
	data := binding.BindUntypedList(&ii)
	container := widget.NewListWithData(
		data,
		func() fyne.CanvasObject {
			return widget.NewLabel("from")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			if entry, err := i.(binding.Untyped).Get(); err != nil {
				log.Println("Failed to Get item")
			} else {
				w := o.(*widget.Label)
				w.SetText(entry.(labelItem).name)
			}
		})

	container.OnSelected = func(id widget.ListItemID) {
		d, _ := data.Get()
		n := d[id].(labelItem).name
		log.Printf("Selected label %v", n)
	}
	return container, data
}

func makeRefreshButton(w fyne.Window, characterID int32) *widget.Button {
	b := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if characterID == 0 {
			info := dialog.NewInformation(
				"Warning",
				"Please select a character first.",
				w,
			)
			info.Show()
			return
		}
		err := core.UpdateMails(characterID)
		if err != nil {
			log.Fatal(err)
		}
	})
	return b
}
