package model

import "time"

// TODO: Maybe combine with CharacterUpdateStatus ??

type CharacterStatus struct {
	CharacterID   int32
	CharacterName string
	ErrorMessage  string
	CompletedAt   time.Time
	Section       CharacterSection
	StartedAt     time.Time
	UpdateAt      time.Time
}

func (cs CharacterStatus) IsOK() bool {
	return cs.ErrorMessage == ""
}

func (cs CharacterStatus) HasData() bool {
	return cs.CharacterID != 0
}

func (cs CharacterStatus) IsCurrent() bool {
	if cs.CompletedAt.IsZero() {
		return false
	}
	return time.Now().Before(cs.CompletedAt.Add(cs.Section.Timeout() * 2))
}

func (cs CharacterStatus) IsMissing() bool {
	return cs.CompletedAt.IsZero()
}

func (cs CharacterStatus) IsRunning() bool {
	return !cs.StartedAt.IsZero()
}
