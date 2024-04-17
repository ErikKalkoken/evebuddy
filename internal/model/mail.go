package model

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
)

// Special mail label IDs
const (
	MailLabelAll      = 1<<31 - 1
	MailLabelNone     = 0
	MailLabelInbox    = 1
	MailLabelSent     = 2
	MailLabelCorp     = 4
	MailLabelAlliance = 8
)

type MailLabel struct {
	ID          int64
	CharacterID int32
	Color       string
	LabelID     int32
	Name        string
	UnreadCount int
}

var bodyPolicy = bluemonday.StrictPolicy()

// An Eve mail belonging to a character.
type Mail struct {
	Body        string
	CharacterID int32
	From        EveEntity
	Labels      []MailLabel
	IsRead      bool
	ID          int64
	MailID      int32
	Recipients  []EveEntity
	Subject     string
	Timestamp   time.Time
}

// BodyPlain returns a mail's body as plain text.
func (m *Mail) BodyPlain() string {
	t := strings.ReplaceAll(m.Body, "<br>", "\n")
	b := html.UnescapeString(bodyPolicy.Sanitize(t))
	return b
}

// BodyForward returns a mail's body for a mail forward or reply.
func (m *Mail) ToString(format string) string {
	s := "\n---\n"
	s += m.MakeHeaderText(format)
	s += "\n\n"
	s += m.BodyPlain()
	return s
}

// MakeHeaderText returns the mail's header as formatted text.
func (m *Mail) MakeHeaderText(format string) string {
	var names []string
	for _, n := range m.Recipients {
		names = append(names, n.Name)
	}
	header := fmt.Sprintf(
		"From: %s\nSent: %s\nTo: %s",
		m.From.Name,
		m.Timestamp.Format(format),
		strings.Join(names, ", "),
	)
	return header
}

// RecipientNames returns the names of the recipients.
func (m *Mail) RecipientNames() []string {
	ss := make([]string, len(m.Recipients))
	for i, r := range m.Recipients {
		ss[i] = r.Name
	}
	return ss
}
