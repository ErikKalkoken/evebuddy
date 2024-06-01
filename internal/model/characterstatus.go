package model

import "time"

type CharacterStatus struct {
	CharacterID   int32
	CharacterName string
	ErrorMessage  string
	LastUpdatedAt time.Time
	Section       CharacterSection
}

func (cs CharacterStatus) IsOK() bool {
	return cs.ErrorMessage == ""
}

func (cs CharacterStatus) IsCurrent() bool {
	if cs.LastUpdatedAt.IsZero() {
		return false
	}
	return time.Now().Before(cs.LastUpdatedAt.Add(cs.Section.Timeout() * 2))
}
