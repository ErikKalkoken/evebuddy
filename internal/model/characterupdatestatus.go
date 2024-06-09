package model

import "time"

type CharacterUpdateStatus struct {
	ID           int64
	CharacterID  int32
	CompletedAt  time.Time
	ContentHash  string
	ErrorMessage string
	Section      CharacterSection
	StartedAt    time.Time
	UpdatedAt    time.Time
}

func (cus CharacterUpdateStatus) IsOK() bool {
	return cus.ErrorMessage == ""
}
