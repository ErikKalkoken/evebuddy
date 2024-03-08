package ui

import (
	"example/esiapp/internal/storage"
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type mails struct {
	container fyne.CanvasObject
}

func (e *esiApp) newMails() *mails {
	mm, err := storage.FetchMailsForLabel(e.characterID, 0)
	if err != nil {
		log.Fatalf("Failed to fetch mail: %v", err)
	}

	mail := newMail()

	headersTotal := widget.NewLabel(fmt.Sprintf("%d mails", len(mm)))
	headersList := widget.NewList(
		func() int {
			return len(mm)
		},
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
		func(i widget.ListItemID, o fyne.CanvasObject) {
			m := mm[i]
			// style := fyne.TextStyle{Bold: !mail.IsRead}
			parent := o.(*fyne.Container)
			top := parent.Objects[0].(*fyne.Container)
			from := top.Objects[0].(*widget.Label)
			from.SetText(m.From.Name)
			timestamp := top.Objects[2].(*widget.Label)
			timestamp.SetText(m.TimeStamp.Format(myDateTime))
			subject := parent.Objects[1].(*widget.Label)
			subject.SetText(m.Subject)
		})
	headersList.OnSelected = func(id widget.ListItemID) {
		m := mm[id]
		mail.update(m)
	}

	headers := container.NewBorder(headersTotal, nil, nil, nil, headersList)

	mainMails := container.NewHSplit(headers, mail.container)
	mainMails.SetOffset(0.35)

	mails := mails{container: mainMails}

	return &mails
}
