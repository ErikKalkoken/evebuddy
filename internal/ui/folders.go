package ui

import (
	"example/esiapp/internal/core"
	"example/esiapp/internal/storage"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type labelItem struct {
	id   int32
	name string
}

type folders struct {
	container     fyne.CanvasObject
	boundList     binding.ExternalUntypedList
	boundCharID   binding.ExternalInt
	headers       *headers
	list          *widget.List
	refreshButton *widget.Button
}

func (f *folders) update(charID int32) {
	if charID == 0 {
		f.refreshButton.Disable()
	} else {
		f.refreshButton.Enable()
	}
	labels, err := storage.FetchAllMailLabels(charID)
	if err != nil {
		log.Fatal(err)
	}
	f.boundCharID.Set(int(charID))
	var ii []interface{}
	if len(labels) > 0 {
		ii = append(ii, labelItem{id: allMailsLabelID, name: "All Mails"})
		for _, l := range labels {
			ii = append(ii, labelItem{id: l.ID, name: l.Name})
		}
	}
	f.boundList.Set(ii)
	f.list.Select(0)
	f.list.ScrollToTop()
	f.headers.update(charID, allMailsLabelID)
}

func (e *esiApp) newFolders(headers *headers) *folders {
	list, boundList, boundCharID := makeFolderList(headers)
	b := makeRefreshButton(boundCharID)
	c := container.NewBorder(b, nil, nil, nil, list)
	f := folders{
		container:     c,
		boundList:     boundList,
		boundCharID:   boundCharID,
		headers:       headers,
		list:          list,
		refreshButton: b,
	}
	return &f
}

func makeFolderList(headers *headers) (*widget.List, binding.ExternalUntypedList, binding.ExternalInt) {
	var ii []interface{}
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
				log.Println("Failed to label item")
				return
			}
			w := o.(*widget.Label)
			w.SetText(entry.(labelItem).name)
		})

	container.OnSelected = func(iID widget.ListItemID) {
		d, err := boundList.Get()
		if err != nil {
			log.Println("Failed to char ID item")
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

func makeRefreshButton(boundCharID binding.ExternalInt) *widget.Button {
	b := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		charID, err := boundCharID.Get()
		if err != nil {
			log.Print("Failed to get character ID")
			return
		}
		if err := core.UpdateMails(int32(charID)); err != nil {
			log.Printf("Failed to update mails for character %d: %v", charID, err)
			return
		}
	})
	return b
}
