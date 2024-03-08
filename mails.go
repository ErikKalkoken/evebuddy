package main

import (
	"example/esiapp/internal/storage"
	"fmt"
	"html"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/microcosm-cc/bluemonday"
)

type mails struct {
	container fyne.CanvasObject
}

func (e *esiApp) newMails() *mails {
	mm, err := storage.FetchMailsForLabel(e.characterID, 0)
	if err != nil {
		log.Fatalf("Failed to fetch mail: %v", err)
	}
	mailHeaderSubject := widget.NewLabel("")
	mailHeaderSubject.TextStyle = fyne.TextStyle{Bold: true}
	mailHeaderSubject.Truncation = fyne.TextTruncateEllipsis
	mailHeaderBlock := widget.NewLabel("")
	mailHeader := container.NewVBox(mailHeaderSubject, mailHeaderBlock)

	mailBody := widget.NewLabel("")
	mailBody.Wrapping = fyne.TextWrapBreak

	detail := container.NewBorder(mailHeader, nil, nil, nil, container.NewVScroll(mailBody))

	headersTotal := widget.NewLabel(fmt.Sprintf("%d mails", len(mm)))
	blue := bluemonday.StrictPolicy()
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
			mail := mm[i]
			// style := fyne.TextStyle{Bold: !mail.IsRead}
			parent := o.(*fyne.Container)
			top := parent.Objects[0].(*fyne.Container)
			from := top.Objects[0].(*widget.Label)
			from.SetText(mail.From.Name)
			timestamp := top.Objects[2].(*widget.Label)
			timestamp.SetText(mail.TimeStamp.Format(myDateTime))
			subject := parent.Objects[1].(*widget.Label)
			subject.SetText(mail.Subject)
		})
	headersList.OnSelected = func(id widget.ListItemID) {
		mail := mm[id]
		mailHeaderSubject.SetText(mail.Subject)
		var names []string
		for _, n := range mail.Recipients {
			names = append(names, n.Name)
		}
		mailHeaderBlock.SetText("From: " + mail.From.Name + "\nSent: " + mail.TimeStamp.Format(myDateTime) + "\nTo: " + strings.Join(names, ", "))
		text := strings.ReplaceAll(mail.Body, "<br>", "\n")
		mailBody.SetText(html.UnescapeString(blue.Sanitize(text)))
	}

	headers := container.NewBorder(headersTotal, nil, nil, nil, headersList)

	mainMails := container.NewHSplit(headers, detail)
	mainMails.SetOffset(0.35)

	mails := mails{container: mainMails}

	return &mails
}
