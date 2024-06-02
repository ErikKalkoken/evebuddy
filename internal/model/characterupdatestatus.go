package model

import "time"

type CharacterUpdateStatus struct {
	ID            int64
	CharacterID   int32
	ErrorMessage  string
	Section       CharacterSection
	LastUpdatedAt time.Time
	ContentHash   string
}

func (cus CharacterUpdateStatus) IsOK() bool {
	return cus.ErrorMessage == ""
}
