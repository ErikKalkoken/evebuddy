package app

import (
	"time"
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
