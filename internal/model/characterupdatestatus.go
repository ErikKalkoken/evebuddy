package model

import "time"

type CharacterUpdateStatus struct {
	ID            int64
	CharacterID   int32
	CharacterName string
	CompletedAt   time.Time
	ContentHash   string
	ErrorMessage  string
	Section       CharacterSection
	StartedAt     time.Time
	UpdatedAt     time.Time
}

func (s CharacterUpdateStatus) IsOK() bool {
	return s.ErrorMessage == ""
}

func (cs CharacterUpdateStatus) HasData() bool {
	return cs.CharacterID != 0
}

func (s CharacterUpdateStatus) IsExpired() bool {
	if s.CompletedAt.IsZero() {
		return true
	}
	timeout := s.Section.Timeout()
	deadline := s.CompletedAt.Add(timeout)
	return time.Now().After(deadline)
}

func (cs CharacterUpdateStatus) IsCurrent() bool {
	if cs.CompletedAt.IsZero() {
		return false
	}
	return time.Now().Before(cs.CompletedAt.Add(cs.Section.Timeout() * 2))
}

func (cs CharacterUpdateStatus) IsMissing() bool {
	return cs.CompletedAt.IsZero()
}

func (cs CharacterUpdateStatus) IsRunning() bool {
	return !cs.StartedAt.IsZero()
}
