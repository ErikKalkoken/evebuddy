package app

import "time"

// Updates status of a general section
type GeneralSectionStatus struct {
	ID           int64
	ContentHash  string
	ErrorMessage string
	CompletedAt  time.Time
	Section      GeneralSection
	StartedAt    time.Time
	UpdatedAt    time.Time
}

func (s GeneralSectionStatus) IsOK() bool {
	return s.ErrorMessage == ""
}

func (s GeneralSectionStatus) IsExpired() bool {
	if s.CompletedAt.IsZero() {
		return true
	}
	timeout := s.Section.Timeout()
	deadline := s.CompletedAt.Add(timeout)
	return time.Now().After(deadline)
}
