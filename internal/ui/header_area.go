package ui

import (
	"example/esiapp/internal/model"
	"fmt"
	"image/color"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type mailItem struct {
	id        uint64
	subject   string
	from      string
	timestamp time.Time
}

// headerArea is the UI area showing the list of mail headers.
type headerArea struct {
	content    fyne.CanvasObject
	list       *widget.List
	boundList  binding.ExternalUntypedList
	boundTotal binding.String
	mailArea   *mailArea
	ui         *ui
}

func (u *ui) NewHeaderArea(mail *mailArea) *headerArea {
	var x []interface{}
	boundList := binding.BindUntypedList(&x)
	list := widget.NewListWithData(
		boundList,
		func() fyne.CanvasObject {
			from := canvas.NewText("xxxxxxxxxxxxxxx", color.White)
			timestamp := canvas.NewText("xxxxxxxxxxxxxxx", color.White)
			subject := canvas.NewText("subject", color.White)
			return container.NewPadded(container.NewPadded(container.NewVBox(
				container.NewHBox(from, layout.NewSpacer(), timestamp),
				subject,
			)))
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			entry, err := i.(binding.Untyped).Get()
			if err != nil {
				slog.Error("Failed to get item")
				return
			}
			m := entry.(mailItem)
			parent := o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container)
			top := parent.Objects[0].(*fyne.Container)

			from := top.Objects[0].(*canvas.Text)
			from.Text = m.from
			from.Refresh()

			timestamp := top.Objects[2].(*canvas.Text)
			timestamp.Text = m.timestamp.Format(myDateTime)
			timestamp.Refresh()

			subject := parent.Objects[1].(*canvas.Text)
			subject.Text = m.subject
			subject.TextStyle = fyne.TextStyle{Bold: true}
			subject.Refresh()
		})

	list.OnSelected = func(id widget.ListItemID) {
		d, err := boundList.Get()
		if err != nil {
			slog.Error("Failed to get mail item", "error", err)
			return
		}
		m := d[id].(mailItem)
		mail.Redraw(m.id)
	}

	boundTotal := binding.NewString()
	total := widget.NewLabelWithData(boundTotal)
	c := container.NewBorder(total, nil, nil, nil, list)

	m := headerArea{
		content:    c,
		list:       list,
		boundList:  boundList,
		boundTotal: boundTotal,
		mailArea:   mail,
		ui:         u,
	}
	return &m
}

func (h *headerArea) Redraw(charID int32, folder node) {
	var d []interface{}
	var mm []model.Mail
	var err error
	switch folder.Category {
	case nodeCategoryLabel:
		mm, err = model.FetchMailsForLabel(charID, folder.ObjID)
	case nodeCategoryList:
		mm, err = model.FetchMailsForList(charID, folder.ObjID)
	}
	if err != nil {
		slog.Error("Failed to fetch mail", "characterID", charID, "error", err)
	} else {
		for _, m := range mm {
			o := mailItem{
				id:        m.ID,
				from:      m.From.Name,
				subject:   m.Subject,
				timestamp: m.Timestamp,
			}
			d = append(d, o)
		}
	}
	h.boundList.Set(d)

	s := fmt.Sprintf("%d mails", len(mm))
	h.boundTotal.Set(s)

	if len(mm) > 0 {
		h.mailArea.Redraw(mm[0].ID)
		h.list.Select(0)
		h.list.ScrollToTop()
	} else {
		h.mailArea.Clear()
	}
}
