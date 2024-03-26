package gui

import (
	"example/esiapp/internal/model"
	"log/slog"

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
	esiApp        *eveApp
	content       fyne.CanvasObject
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

// func (f *folders) updateMailsWithID(charID int32) {
// 	err := f.boundCharID.Set(int(charID))
// 	if err != nil {
// 		slog.Error("Failed to set char ID", "error", err)
// 	} else {
// 		f.updateMails()
// 	}
// }

func (f *folders) updateMails() {
	charID, err := f.boundCharID.Get()
	if err != nil {
		slog.Error("Failed to get character ID", "erro", err)
		return
	}
	status := f.esiApp.statusBar
	go func() {
		err = UpdateMails(int32(charID), status)
		if err != nil {
			status.setText("Failed to fetch mail")
			slog.Error("Failed to update mails", "characterID", charID, "error", err)
			return
		}
		f.update(int32(charID))
	}()
}

func (f *folders) update(charID int32) {
	if charID == 0 {
		f.refreshButton.Disable()
	} else {
		f.refreshButton.Enable()
	}
	if err := f.boundCharID.Set(int(charID)); err != nil {
		slog.Error("Failed to set char ID", "characterID", charID, "error", err)
	}

	var ii []interface{}
	labels, err := model.FetchAllMailLabels(charID)
	if err != nil {
		slog.Error("Failed to fetch mail labels", "characterID", charID, "error", err)
	} else {
		if len(labels) > 0 {
			ii = append(ii, labelItem{id: model.AllMailsLabelID, name: "All Mails"})
			for _, l := range labels {
				ii = append(ii, labelItem{id: l.LabelID, name: l.Name})
			}
		}
	}
	f.boundList.Set(ii)
	f.list.Select(0)
	f.list.ScrollToTop()
	f.headers.update(charID, model.AllMailsLabelID)
}

func (e *eveApp) newFolders(headers *headers) *folders {
	list, boundList, boundCharID := makeFolderList(headers)
	f := folders{
		esiApp:      e,
		boundList:   boundList,
		boundCharID: boundCharID,
		headers:     headers,
		list:        list,
	}
	f.addRefreshButton()
	b := widget.NewButtonWithIcon("New message", theme.ContentAddIcon(), func() {
		d := dialog.NewInformation("New message", "PLACEHOLDER", e.winMain)
		d.Show()
	})
	top := container.NewHBox(f.refreshButton, b)
	c := container.NewBorder(top, nil, nil, nil, f.list)
	f.content = c
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
				slog.Error("Failed to get label", "error", err)
				return
			}
			w := o.(*widget.Label)
			w.SetText(entry.(labelItem).name)
		})

	container.OnSelected = func(iID widget.ListItemID) {
		d, err := boundList.Get()
		if err != nil {
			slog.Error("Failed to char ID item", "error", err)
			return
		}
		n := d[iID].(labelItem)
		cID, err := boundCharID.Get()
		if err != nil {
			slog.Error("Failed to get item", "error", err)
			return
		}
		headers.update(int32(cID), n.id)

	}
	return container, boundList, boundCharID
}
