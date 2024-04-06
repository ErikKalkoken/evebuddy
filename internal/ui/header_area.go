package ui

import (
	"example/esiapp/internal/model"
	"image/color"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// headerArea is the UI area showing the list of mail headers.
type headerArea struct {
	listData      binding.IntList
	total         binding.Int
	content       fyne.CanvasObject
	currentFolder node
	list          *widget.List
	ui            *ui
}

func (u *ui) NewHeaderArea() *headerArea {
	listData := binding.NewIntList()
	list := widget.NewListWithData(
		listData,
		func() fyne.CanvasObject {
			from := canvas.NewText("xxxxxxxxxxxxxxx", color.White)
			timestamp := canvas.NewText("xxxxxxxxxxxxxxx", color.White)
			subject := canvas.NewText("subject", color.White)
			return container.NewPadded(container.NewPadded(container.NewVBox(
				container.NewHBox(from, layout.NewSpacer(), timestamp),
				subject,
			)))
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			b := di.(binding.Int)
			mailID, err := b.Get()
			if err != nil {
				slog.Error("Failed to get item")
				return
			}
			m, err := model.FetchMail(uint64(mailID))
			if err != nil {
				slog.Error("Failed to get mail")
				return
			}
			parent := co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container)
			top := parent.Objects[0].(*fyne.Container)

			from := top.Objects[0].(*canvas.Text)
			from.Text = m.From.Name
			from.TextStyle = fyne.TextStyle{Bold: !m.IsRead}
			from.Refresh()

			timestamp := top.Objects[2].(*canvas.Text)
			timestamp.Text = m.Timestamp.Format(myDateTime)
			timestamp.TextStyle = fyne.TextStyle{Bold: !m.IsRead}
			timestamp.Refresh()

			subject := parent.Objects[1].(*canvas.Text)
			subject.Text = m.Subject
			subject.TextStyle = fyne.TextStyle{Bold: !m.IsRead}
			subject.Refresh()
		})

	list.OnSelected = func(id widget.ListItemID) {
		di, err := listData.GetItem(id)
		if err != nil {
			slog.Error("Failed to get data item", "error", err)
			return
		}
		b := di.(binding.Int)
		mailID, err := b.Get()
		if err != nil {
			slog.Error("Failed to get item")
			return
		}
		u.mailArea.Redraw(uint64(mailID), id)
	}

	total := binding.NewInt()
	totalStr := binding.IntToStringWithFormat(total, "%d mails")
	label := widget.NewLabelWithData(totalStr)
	c := container.NewBorder(label, nil, nil, nil, list)

	m := headerArea{
		content:  c,
		list:     list,
		listData: listData,
		total:    total,
		ui:       u,
	}
	return &m
}

func (h *headerArea) RedrawCurrent() {
	h.redraw(h.currentFolder)
}

func (h *headerArea) Redraw(folder node) {
	h.redraw(folder)
}

func (h *headerArea) redraw(folder node) {
	var mm []model.Mail
	var err error
	charID := h.ui.CurrentCharID()
	switch folder.Category {
	case nodeCategoryLabel:
		mm, err = model.FetchMailsForLabel(charID, folder.ObjID)
	case nodeCategoryList:
		mm, err = model.FetchMailsForList(charID, folder.ObjID)
	}
	var mailIDs []int
	if err != nil {
		slog.Error("Failed to fetch mail", "characterID", charID, "error", err)
	} else {
		for _, m := range mm {
			mailIDs = append(mailIDs, int(m.ID))
		}
	}
	h.listData.Set(mailIDs)
	h.currentFolder = folder
	h.total.Set(len(mm))

	if len(mm) > 0 {
		h.ui.mailArea.Redraw(mm[0].ID, 0)
		h.list.Select(0)
		h.list.ScrollToTop()
	} else {
		h.ui.mailArea.Clear()
	}
}
