package model

import "time"

type CharacterSectionStatus struct {
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

func (s CharacterSectionStatus) IsOK() bool {
	return s.ErrorMessage == ""
}

func (s CharacterSectionStatus) HasData() bool {
	return s.CharacterID != 0
}

func (s CharacterSectionStatus) IsExpired() bool {
	if s.CompletedAt.IsZero() {
		return true
	}
	timeout := s.Section.Timeout()
	deadline := s.CompletedAt.Add(timeout)
	return time.Now().After(deadline)
}

func (s CharacterSectionStatus) IsCurrent() bool {
	if s.CompletedAt.IsZero() {
		return false
	}
	return time.Now().Before(s.CompletedAt.Add(s.Section.Timeout() * 2))
}

func (s CharacterSectionStatus) IsMissing() bool {
	return s.CompletedAt.IsZero()
}

func (s CharacterSectionStatus) IsRunning() bool {
	return !s.StartedAt.IsZero()
}
