package app

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"bytes"

	"github.com/yuin/goldmark"

	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CharacterNotification struct {
	ID             int64
	Body           optional.Optional[string] // generated body text in markdown
	CharacterID    int32
	IsProcessed    bool
	IsRead         bool
	NotificationID int64
	RecipientName  string // TODO: Replace with EveEntity
	Sender         *EveEntity
	Text           string
	Timestamp      time.Time
	Title          optional.Optional[string] // generated title text in markdown
	Type           string                    // This is a string, so that it can handle unknown types
}

// TitleDisplay returns the rendered title when it exists or else the fake tile.
func (cn *CharacterNotification) TitleDisplay() string {
	if cn.Title.IsEmpty() {
		return cn.TitleFake()
	}
	return cn.Title.ValueOrZero()
}

// TitleFake returns a title for output made from the name of the type.
func (cn *CharacterNotification) TitleFake() string {
	var b strings.Builder
	var last rune
	for _, r := range cn.Type {
		if unicode.IsUpper(r) && unicode.IsLower(last) {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
		last = r
	}
	return b.String()
}

// Header returns the header of a notification.
func (cn *CharacterNotification) Header() string {
	s := fmt.Sprintf(
		"From: %s\n"+
			"Sent: %s",
		cn.Sender.Name,
		cn.Timestamp.Format(TimeDefaultFormat),
	)
	if cn.RecipientName != "" {
		s += fmt.Sprintf("\nTo: %s", cn.RecipientName)
	}
	return s
}

// String returns the content of a notification as string.
func (cn *CharacterNotification) String() string {
	s := cn.TitleDisplay() + "\n" + cn.Header()
	b, err := cn.BodyPlain()
	if err != nil {
		slog.Error("render notification to string", "id", cn.ID, "error", err)
		return s
	}
	s += "\n\n"
	if b.IsEmpty() {
		s += "(no body)"
	} else {
		s += b.ValueOrZero()
	}
	return s
}

// BodyPlain returns the body of a notification as plain text.
func (cn *CharacterNotification) BodyPlain() (optional.Optional[string], error) {
	var b optional.Optional[string]
	if cn.Body.IsEmpty() {
		return b, nil
	}
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(cn.Body.ValueOrZero()), &buf); err != nil {
		return b, fmt.Errorf("convert notification body: %w", err)
	}
	b.Set(evehtml.Strip(buf.String()))
	return b, nil
}
