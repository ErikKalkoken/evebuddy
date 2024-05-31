package model

import "time"

type CharacterStatus struct {
	ErrorMessage  string
	LastUpdatedAt time.Time
	Section       string
	Timeout       time.Duration
}

func (s *CharacterStatus) IsOK() bool {
	return s.ErrorMessage == ""
}

func (s *CharacterStatus) IsCurrent() bool {
	if s.LastUpdatedAt.IsZero() {
		return false
	}
	return time.Now().Before(s.LastUpdatedAt.Add(s.Timeout * 2))
}
