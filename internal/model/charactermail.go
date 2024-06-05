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

// An Eve mail header belonging to a character.
type CharacterMailHeader struct {
	CharacterID int32
	From        string
	IsRead      bool
	MailID      int32
	Subject     string
	Timestamp   time.Time
}

// BodyPlain returns a mail's body as plain text.
func (cm CharacterMail) BodyPlain() string {
	return converter.EveHTMLToPlain(cm.Body)
}

// BodyForward returns a mail's body for a mail forward or reply.
func (cm CharacterMail) ToString(format string) string {
	s := "\n---\n"
	s += cm.MakeHeaderText(format)
	s += "\n\n"
	s += cm.BodyPlain()
	return s
}

// MakeHeaderText returns the mail's header as formatted text.
func (cm CharacterMail) MakeHeaderText(format string) string {
	var names []string
	for _, n := range cm.Recipients {
		names = append(names, n.Name)
	}
	header := fmt.Sprintf(
		"From: %s\nSent: %s\nTo: %s",
		cm.From.Name,
		cm.Timestamp.Format(format),
		strings.Join(names, ", "),
	)
	return header
}

// RecipientNames returns the names of the recipients.
func (cm CharacterMail) RecipientNames() []string {
	ss := make([]string, len(cm.Recipients))
	for i, r := range cm.Recipients {
		ss[i] = r.Name
	}
	return ss
}

func (cm CharacterMail) BodyToMarkdown() string {
	return converter.EveHTMLtoMarkdown(cm.Body)
}
