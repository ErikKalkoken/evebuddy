package app

import (
	"strings"
	"time"
	"unicode"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CharacterNotification struct {
	ID             int64
	Body           optional.Optional[string]
	CharacterID    int32
	IsRead         bool
	NotificationID int64
	Sender         *EveEntity
	Text           string
	Timestamp      time.Time
	Title          optional.Optional[string]
	Type           string
}

// TitleOutput returns the rendered title when it exists, or else the fake tile.
func (cn *CharacterNotification) TitleOutput() string {
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