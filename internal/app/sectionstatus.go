package app

import "time"

// A SectionStatus represents the latest status of an update from ESI for a section.
// This type contains the shared functionality among all kinds of section status.
type SectionStatus struct {
	CompletedAt  time.Time
	ContentHash  string
	ErrorMessage string
	Section      section
	StartedAt    time.Time
	UpdatedAt    time.Time
}

// HasContent reports whether a section has data.
func (s SectionStatus) HasContent() bool {
	return s.ContentHash != ""
}

// HasError reports whether the last update of a section resulted in an error.
func (s SectionStatus) HasError() bool {
	return s.ErrorMessage != ""
}

// WasUpdatedWithinErrorTimedOut reports whether the last update was within the error timeout.
func (s SectionStatus) WasUpdatedWithinErrorTimedOut() bool {
	x := time.Since(s.UpdatedAt)
	return x > sectionErrorTimeout
}

// IsMissing reports whether this section has ever been successfully updated.
func (s SectionStatus) IsMissing() bool {
	return s.CompletedAt.IsZero()
}

func (s SectionStatus) IsExpired() bool {
	if s.CompletedAt.IsZero() {
		return true
	}
	timeout := s.Section.Timeout()
	deadline := s.CompletedAt.Add(timeout)
	return time.Now().After(deadline)
}

// CharacterSectionStatus represents the status for a character's section.
type CharacterSectionStatus struct {
	SectionStatus
	CharacterID   int32
	CharacterName string
}

type CorporationSectionStatus struct {
	SectionStatus
	Comment         string
	CorporationID   int32
	CorporationName string
}

// GeneralSectionStatus represents the status of a general section.
type GeneralSectionStatus struct {
	SectionStatus
}
