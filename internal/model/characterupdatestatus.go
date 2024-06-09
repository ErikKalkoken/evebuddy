package model

import "time"

type CharacterUpdateStatus struct {
	ID           int64
	CharacterID  int32
	ErrorMessage string
	Section      CharacterSection
	StartedAt    time.Time
	CompletedAt  time.Time
	ContentHash  string
}

func (cus CharacterUpdateStatus) IsOK() bool {
	return cus.ErrorMessage == ""
}
