package app

import (
	"strings"
	"time"
	"unicode"
)

type CharacterNotification struct {
	ID             int64
	CharacterID    int32
	IsRead         bool
	NotificationID int64
	Sender         *EveEntity
	Text           string
	Timestamp      time.Time
	Type           string
}

func (cn *CharacterNotification) Title() string {
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
