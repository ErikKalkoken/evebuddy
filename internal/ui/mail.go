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
	content *fyne.Container
	policy  *bluemonday.Policy
}

func (m *mail) clear() {
	m.content.RemoveAll()
	m.content.Refresh()
}

func (m *mail) update(mailID uint) {
	m.content.RemoveAll()
	defer m.content.Refresh()

	mail, err := storage.FetchMailByID(mailID)
	if err != nil {
		log.Printf("Failed to fetch mail %v: %v", mailID, err)
		return
	}

	subject := widget.NewLabel(mail.Subject)
	subject.TextStyle = fyne.TextStyle{Bold: true}
	subject.Truncation = fyne.TextTruncateEllipsis

	var names []string
	for _, n := range mail.Recipients {
		names = append(names, n.Name)
	}
	t := fmt.Sprintf(
		"From: %s\nSent: %s\nTo: %s",
		mail.From.Name,
		mail.TimeStamp.Format(myDateTime),
		strings.Join(names, ", "),
	)
	header := widget.NewLabel(t)
	header.Wrapping = fyne.TextWrapBreak

	wrapper := container.NewVBox(subject, header)

	text := strings.ReplaceAll(mail.Body, "<br>", "\n")
	body := widget.NewLabel(html.UnescapeString(m.policy.Sanitize(text)))
	body.Wrapping = fyne.TextWrapBreak
	bodyWithScroll := container.NewVScroll(body)

	inner := container.NewBorder(wrapper, nil, nil, nil, bodyWithScroll)

	m.content.Add(inner)
}

func (e *esiApp) newMail() *mail {
	policy := bluemonday.StrictPolicy()
	c := container.NewStack()
	m := mail{
		content: c,
		policy:  policy,
	}
	return &m
}
