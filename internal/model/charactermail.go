package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/eveonline/converter"
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

// A mail label for an Eve mail belonging to a character.
type CharacterMailLabel struct {
	ID          int64
	CharacterID int32
	Color       string
	LabelID     int32
	Name        string
	UnreadCount int
}

// An Eve mail belonging to a character.
type CharacterMail struct {
	Body        string
	CharacterID int32
	From        *EveEntity
	Labels      []*CharacterMailLabel
	IsRead      bool
	ID          int64
	MailID      int32
	Recipients  []*EveEntity
	Subject     string
	Timestamp   time.Time
}

// BodyPlain returns a mail's body as plain text.
func (m *CharacterMail) BodyPlain() string {
	return converter.XMLToPlain(m.Body)
}

// BodyForward returns a mail's body for a mail forward or reply.
func (m *CharacterMail) ToString(format string) string {
	s := "\n---\n"
	s += m.MakeHeaderText(format)
	s += "\n\n"
	s += m.BodyPlain()
	return s
}

// MakeHeaderText returns the mail's header as formatted text.
func (m *CharacterMail) MakeHeaderText(format string) string {
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
func (m *CharacterMail) RecipientNames() []string {
	ss := make([]string, len(m.Recipients))
	for i, r := range m.Recipients {
		ss[i] = r.Name
	}
	return ss
}

func (m *CharacterMail) BodyToMarkdown() string {
	return converter.XMLtoMarkdown(m.Body)
}
