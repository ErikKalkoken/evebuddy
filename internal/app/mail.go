package app

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
)

type SendMailMode uint

const (
	SendMailNew SendMailMode = iota + 1
	SendMailReply
	SendMailReplyAll
	SendMailForward
)

// Special mail label IDs
const (
	MailLabelAll      = 1<<31 - 1
	MailLabelUnread   = 1<<31 - 2
	MailLabelNone     = 0
	MailLabelInbox    = 1
	MailLabelSent     = 2
	MailLabelCorp     = 4
	MailLabelAlliance = 8
)

// CharacterMailLabel is a mail label for an EVE mail belonging to a character.
type CharacterMailLabel struct {
	ID          int64
	CharacterID int32
	Color       string
	LabelID     int32
	Name        string
	UnreadCount int
}

// CharacterMail is an EVE mail belonging to a character.
type CharacterMail struct {
	Body        string
	CharacterID int32
	From        *EveEntity
	Labels      []*CharacterMailLabel
	IsProcessed bool
	IsRead      bool
	ID          int64
	MailID      int32
	Recipients  []*EveEntity
	Subject     string
	Timestamp   time.Time
}

// CharacterMailHeader is an EVE mail header belonging to a character.
type CharacterMailHeader struct {
	CharacterID int32
	From        *EveEntity
	IsRead      bool
	ID          int64
	MailID      int32
	Subject     string
	Timestamp   time.Time
}

// BodyPlain returns a mail's body as plain text.
func (cm CharacterMail) BodyPlain() string {
	return evehtml.ToPlain(cm.Body)
}

// String returns a mail's content as string.
func (cm CharacterMail) String() string {
	s := fmt.Sprintf("%s\n", cm.Subject) + cm.Header() + "\n\n" + cm.BodyPlain()
	return s
}

// Header returns a mail's header as string.
func (cm CharacterMail) Header() string {
	var names []string
	for _, n := range cm.Recipients {
		names = append(names, n.Name)
	}
	header := fmt.Sprintf(
		"From: %s\n"+
			"Sent: %s\n"+
			"To: %s",
		cm.From.Name,
		cm.Timestamp.Format(DateTimeFormat),
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
	s, err := evehtml.ToMarkdown(cm.Body)
	if err != nil {
		slog.Error("Failed to convert mail body to markdown", "characterID", cm.CharacterID, "mailID", cm.MailID, "error", err)
		return ""
	}
	return s
}
