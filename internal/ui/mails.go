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

type mails struct {
	container  fyne.CanvasObject
	boundList  binding.ExternalUntypedList
	boundTotal binding.String
}

func (e *esiApp) newMails() *mails {

	mail := newMail()

	var x []interface{}
	boundList := binding.BindUntypedList(&x)
	headersList := widget.NewListWithData(
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

	headersList.OnSelected = func(id widget.ListItemID) {
		d, err := boundList.Get()
		if err != nil {
			log.Println("Failed to Get item")
			return
		}
		m := d[id].(mailItem)
		mail.update(m.id)
	}

	boundTotal := binding.NewString()
	total := widget.NewLabelWithData(boundTotal)
	headers := container.NewBorder(total, nil, nil, nil, headersList)

	mailsC := container.NewHSplit(headers, mail.container)
	mailsC.SetOffset(0.35)

	m := mails{
		container:  mailsC,
		boundList:  boundList,
		boundTotal: boundTotal,
	}
	return &m
}

func (m *mails) update(charID int32, labelID int32) {
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
	m.boundList.Set(d)

	s := fmt.Sprintf("%d mails", len(mm))
	m.boundTotal.Set(s)
}
