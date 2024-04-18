package ui

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	islices "example/evebuddy/internal/helper/slices"
	"example/evebuddy/internal/storage"
)

// headerArea is the UI area showing the list of mail headers.
type headerArea struct {
	listData      binding.IntList // list of character's mail IDs
	infoText      binding.String
	content       fyne.CanvasObject
	currentFolder node
	list          *widget.List
	ui            *ui
}

func (u *ui) NewHeaderArea() *headerArea {
	foregroundColor := theme.ForegroundColor()
	subjectSize := theme.TextSize() * 1.15
	listData := binding.NewIntList()
	list := widget.NewListWithData(
		listData,
		func() fyne.CanvasObject {
			from := canvas.NewText("xxxxxxxxxxxxxxx", foregroundColor)
			timestamp := canvas.NewText("xxxxxxxxxxxxxxx", foregroundColor)
			subject := canvas.NewText("subject", foregroundColor)
			subject.TextSize = subjectSize
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
			characterID := u.CurrentCharID()
			if characterID == 0 {
				return
			}
			m, err := u.service.GetMail(characterID, int32(mailID))
			if err != nil {
				if !errors.Is(err, storage.ErrNotFound) {
					slog.Error("Failed to get mail", "error", err)
				}
				return
			}
			parent := co.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container)
			top := parent.Objects[0].(*fyne.Container)

			from := top.Objects[0].(*canvas.Text)
			var t string
			if u.headerArea != nil && u.headerArea.currentFolder.isSent() {
				t = strings.Join(m.RecipientNames(), ", ")
			} else {
				t = m.From.Name
			}
			from.Text = t
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
		u.mailArea.Redraw(int32(mailID), id)
	}

	infoText := binding.NewString()
	label := widget.NewLabelWithData(infoText)
	c := container.NewBorder(label, nil, nil, nil, list)

	m := headerArea{
		content:  c,
		list:     list,
		listData: listData,
		infoText: infoText,
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
	var mailIDs []int32
	var err error
	characterID := h.ui.CurrentCharID()
	switch folder.Category {
	case nodeCategoryLabel:
		mailIDs, err = h.ui.service.ListMailIDsForLabelOrdered(characterID, folder.ObjID)
	case nodeCategoryList:
		mailIDs, err = h.ui.service.ListMailIDsForListOrdered(characterID, folder.ObjID)
	}
	if err != nil {
		slog.Error("Failed to fetch mail", "characterID", characterID, "error", err)
	}
	ids := islices.ConvertNumeric[int32, int](mailIDs)
	h.listData.Set(ids)
	h.currentFolder = folder
	s := "?"
	updatedAt, err := h.ui.service.MailUpdatedAt(characterID)
	if err != nil {
		slog.Error("Failed to fetch mail update at: %s", err)
	} else {
		if !updatedAt.IsZero() {
			s = humanize.Time(updatedAt)
		}
	}
	h.infoText.Set(fmt.Sprintf("%d mails (%s)", len(mailIDs), s))

	if len(mailIDs) > 0 {
		h.ui.mailArea.Redraw(mailIDs[0], 0)
		h.list.Select(0)
		h.list.ScrollToTop()
	} else {
		h.ui.mailArea.Clear()
	}
}
