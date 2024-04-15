package repository

import (
	"example/evebuddy/internal/repository/sqlc"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
)

var bodyPolicy = bluemonday.StrictPolicy()

// An Eve mail belonging to a character
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

func mailFromDBModel(mail sqlc.Mail, from sqlc.EveEntity, labels []sqlc.MailLabel, recipients []sqlc.EveEntity) Mail {
	if mail.CharacterID == 0 {
		panic("missing character ID")
	}
	var ll []MailLabel
	for _, l := range labels {
		ll = append(ll, mailLabelFromDBModel(l))
	}
	var rr []EveEntity
	for _, r := range recipients {
		rr = append(rr, eveEntityFromDBModel(r))
	}
	m := Mail{
		Body:        mail.Body,
		CharacterID: int32(mail.CharacterID),
		From:        eveEntityFromDBModel(from),
		IsRead:      mail.IsRead,
		ID:          mail.ID,
		Labels:      ll,
		MailID:      int32(mail.MailID),
		Recipients:  rr,
		Subject:     mail.Subject,
		Timestamp:   mail.Timestamp,
	}
	return m
}

// BodyPlain returns a mail's body as plain text.
func (m *Mail) BodyPlain() string {
	t := strings.ReplaceAll(m.Body, "<br>", "\n")
	b := html.UnescapeString(bodyPolicy.Sanitize(t))
	return b
}

// BodyForward returns a mail's body for a mail forward or reply
func (m *Mail) ToString(format string) string {
	s := "\n---\n"
	s += m.MakeHeaderText(format)
	s += "\n\n"
	s += m.BodyPlain()
	return s
}

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

func (m *Mail) RecipientNames() []string {
	ss := make([]string, len(m.Recipients))
	for i, r := range m.Recipients {
		ss[i] = r.Name
	}
	return ss
}
