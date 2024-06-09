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

func (s CharacterUpdateStatus) IsOK() bool {
	return s.ErrorMessage == ""
}

func (s CharacterUpdateStatus) IsExpired() bool {
	if s.CompletedAt.IsZero() {
		return true
	}
	timeout := s.Section.Timeout()
	deadline := s.CompletedAt.Add(timeout)
	return time.Now().After(deadline)
}
