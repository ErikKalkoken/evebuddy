package gui

import (
	"example/esiapp/internal/storage"
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
	id        uint
	subject   string
	from      string
	timestamp time.Time
}

type headers struct {
	content    fyne.CanvasObject
	list       *widget.List
	boundList  binding.ExternalUntypedList
	boundTotal binding.String
	mail       *mail
}

func (h *headers) update(charID int32, labelID int32) {
	var d []interface{}
	mm, err := storage.FetchMailsForLabel(charID, labelID)
	if err != nil {
		slog.Error("Failed to fetch mail", "characterID", charID, "error", err)
	} else {
		for _, m := range mm {
			o := mailItem{
				id:        m.ID,
				from:      m.From.Name,
				subject:   m.Subject,
				timestamp: m.TimeStamp,
			}
			d = append(d, o)
		}
	}
	h.boundList.Set(d)

	s := fmt.Sprintf("%d mails", len(mm))
	h.boundTotal.Set(s)

	if len(mm) > 0 {
		h.mail.update(mm[0].ID)
		h.list.Select(0)
		h.list.ScrollToTop()
	} else {
		h.mail.clear()
	}
}

func (e *esiApp) newHeaders(mail *mail) *headers {
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
		mail.update(m.id)
	}

	boundTotal := binding.NewString()
	total := widget.NewLabelWithData(boundTotal)
	c := container.NewBorder(total, nil, nil, nil, list)

	m := headers{
		content:    c,
		list:       list,
		boundList:  boundList,
		boundTotal: boundTotal,
		mail:       mail,
	}
	return &m
}