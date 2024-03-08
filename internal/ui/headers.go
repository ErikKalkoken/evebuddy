package ui

import (
	"example/esiapp/internal/storage"
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
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
	container  fyne.CanvasObject
	list       *widget.List
	boundList  binding.ExternalUntypedList
	boundTotal binding.String
	mail       *mail
}

func (h *headers) update(charID int32, labelID int32) {
	mm, err := storage.FetchMailsForLabel(charID, labelID)
	if err != nil {
		log.Fatalf("Failed to fetch mail: %v", err)
	}
	var d []interface{}
	for _, m := range mm {
		o := mailItem{
			id:        m.ID,
			from:      m.From.Name,
			subject:   m.Subject,
			timestamp: m.TimeStamp,
		}
		d = append(d, o)
	}
	h.boundList.Set(d)

	s := fmt.Sprintf("%d mails", len(mm))
	h.boundTotal.Set(s)

	if len(mm) > 0 {
		h.mail.update(mm[0].ID)
		h.list.Select(0)
	}
}

func (e *esiApp) newHeaders(mail *mail) *headers {
	var x []interface{}
	boundList := binding.BindUntypedList(&x)
	list := widget.NewListWithData(
		boundList,
		func() fyne.CanvasObject {
			from := widget.NewLabel("from")
			timestamp := widget.NewLabel("timestamp")
			subject := widget.NewLabel("subject")
			subject.TextStyle = fyne.TextStyle{Bold: true}
			subject.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(
				container.NewHBox(from, layout.NewSpacer(), timestamp),
				subject,
			)
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			entry, err := i.(binding.Untyped).Get()
			if err != nil {
				log.Println("Failed to Get item")
				return
			}
			m := entry.(mailItem)
			parent := o.(*fyne.Container)
			top := parent.Objects[0].(*fyne.Container)
			from := top.Objects[0].(*widget.Label)
			from.SetText(m.from)
			timestamp := top.Objects[2].(*widget.Label)
			timestamp.SetText(m.timestamp.Format(myDateTime))
			subject := parent.Objects[1].(*widget.Label)
			subject.SetText(m.subject)
		})

	list.OnSelected = func(id widget.ListItemID) {
		d, err := boundList.Get()
		if err != nil {
			log.Println("Failed to mail item")
			return
		}
		m := d[id].(mailItem)
		mail.update(m.id)
	}

	boundTotal := binding.NewString()
	total := widget.NewLabelWithData(boundTotal)
	c := container.NewBorder(total, nil, nil, nil, list)

	m := headers{
		container:  c,
		list:       list,
		boundList:  boundList,
		boundTotal: boundTotal,
		mail:       mail,
	}
	return &m
}
