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
	esiApp        *esiApp
	container     fyne.CanvasObject
	boundList     binding.ExternalUntypedList
	boundCharID   binding.ExternalInt
	headers       *headers
	list          *widget.List
	refreshButton *widget.Button
}

func (f *folders) addRefreshButton() {
	b := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		f.updateMails()
	})
	f.refreshButton = b
}

func (f *folders) updateMailsWithID(charID int32) {
	err := f.boundCharID.Set(int(charID))
	if err != nil {
		log.Printf("Failed to set char ID: %v", err)
	}
	f.updateMails()
}

func (f *folders) updateMails() {
	charID, err := f.boundCharID.Get()
	if err != nil {
		log.Print("Failed to get character ID")
		return
	}
	p := widget.NewProgressBarInfinite()
	l := widget.NewLabel("Updating mails...")
	c := container.NewVBox(l, p)
	d := dialog.NewCustomWithoutButtons(
		"Refresh mails",
		c,
		f.esiApp.main,
	)
	d.Show()
	err = core.UpdateMails(int32(charID))
	d.Hide()
	if err != nil {
		log.Printf("Failed to update mails for character %d: %v", charID, err)
		return
	}
	f.update(int32(charID))
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
	err = f.boundCharID.Set(int(charID))
	if err != nil {
		log.Printf("Failed to set char ID: %v", err)
	}
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
	f := folders{
		esiApp:      e,
		boundList:   boundList,
		boundCharID: boundCharID,
		headers:     headers,
		list:        list,
	}
	f.addRefreshButton()
	c := container.NewBorder(f.refreshButton, nil, nil, nil, f.list)
	f.container = c
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
