package ui

import (
	"example/esiapp/internal/storage"
	"fmt"
	"html"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/microcosm-cc/bluemonday"
)

type mail struct {
	container fyne.CanvasObject
	subject   *widget.Label
	header    *widget.Label
	body      *widget.Label
	policy    *bluemonday.Policy
}

func (m *mail) update(mailID uint) {
	mail, err := storage.FetchMailByID(mailID)
	if err != nil {
		log.Printf("Failed to render mail: %v", err)
		return
	}

	m.subject.SetText(mail.Subject)
	var names []string
	for _, n := range mail.Recipients {
		names = append(names, n.Name)
	}
	t := fmt.Sprintf(
		"From: %s\nSent:%s\nTo:%s",
		mail.From.Name,
		mail.TimeStamp.Format(myDateTime),
		strings.Join(names, ", "),
	)
	m.header.SetText(t)
	text := strings.ReplaceAll(mail.Body, "<br>", "\n")
	m.body.SetText(html.UnescapeString(m.policy.Sanitize(text)))
}

func (e *esiApp) newMail() *mail {
	subject := widget.NewLabel("")
	subject.TextStyle = fyne.TextStyle{Bold: true}
	subject.Truncation = fyne.TextTruncateEllipsis
	header := widget.NewLabel("")
	wrapper := container.NewVBox(subject, header)

	body := widget.NewLabel("")
	body.Wrapping = fyne.TextWrapBreak
	c := container.NewBorder(wrapper, nil, nil, nil, container.NewVScroll(body))
	policy := bluemonday.StrictPolicy()
	m := mail{
		container: c,
		subject:   subject,
		header:    header,
		body:      body,
		policy:    policy,
	}
	return &m
}
