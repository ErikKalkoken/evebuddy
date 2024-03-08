package ui

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
	container   fyne.CanvasObject
	boundList   binding.ExternalUntypedList
	boundCharID binding.ExternalInt
}

func (f *folders) update(charID int32) {
	labels, err := storage.FetchAllMailLabels(charID)
	if err != nil {
		log.Fatal(err)
	}
	f.boundCharID.Set(int(charID))
	for _, l := range labels {
		f.boundList.Append(labelItem{id: l.ID, name: l.Name})
	}

}

func (e *esiApp) newFolders(headers *headers) *folders {
	list, boundList, boundCharID := makeFolderList(headers)
	button := makeRefreshButton(e.main, boundCharID)
	c := container.NewBorder(button, nil, nil, nil, list)
	f := folders{container: c, boundList: boundList, boundCharID: boundCharID}
	return &f
}

func makeFolderList(headers *headers) (*widget.List, binding.ExternalUntypedList, binding.ExternalInt) {
	var ii []interface{}
	ii = append(ii, labelItem{id: 0, name: "All Mails"})
	boundList := binding.BindUntypedList(&ii)

	var charID int
	boundCharID := binding.BindInt(&charID)

	container := widget.NewListWithData(
		boundList,
		func() fyne.CanvasObject {
			return widget.NewLabel("from")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			entry, err := i.(binding.Untyped).Get()
			if err != nil {
				log.Println("Failed to Get item")
				return
			}
			w := o.(*widget.Label)
			w.SetText(entry.(labelItem).name)
		})

	container.OnSelected = func(iID widget.ListItemID) {
		d, err := boundList.Get()
		if err != nil {
			log.Println("Failed to Get item")
			return
		}
		n := d[iID].(labelItem)
		cID, err := boundCharID.Get()
		if err != nil {
			log.Println("Failed to Get item")
			return
		}
		headers.update(int32(cID), n.id)

	}
	return container, boundList, boundCharID
}

func makeRefreshButton(w fyne.Window, boundCharID binding.ExternalInt) *widget.Button {
	b := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		charID, err := boundCharID.Get()
		if err != nil {
			log.Fatal(err)
		}
		if charID == 0 {
			info := dialog.NewInformation(
				"Warning",
				"Please select a character first.",
				w,
			)
			info.Show()
			return
		}
		if err := core.UpdateMails(int32(charID)); err != nil {
			log.Fatal(err)
		}
	})
	return b
}
