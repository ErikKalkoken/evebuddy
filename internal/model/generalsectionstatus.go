package model

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

func (s GeneralSectionStatus) IsCurrent() bool {
	if s.CompletedAt.IsZero() {
		return false
	}
	return time.Now().Before(s.CompletedAt.Add(s.Section.Timeout() * 2))
}

func (s GeneralSectionStatus) IsMissing() bool {
	return s.CompletedAt.IsZero()
}

func (s GeneralSectionStatus) IsRunning() bool {
	return !s.StartedAt.IsZero()
}
