package model

import "time"

type SectionStatus struct {
	EntityID     int32
	EntityName   string
	CompletedAt  time.Time
	ContentHash  string
	ErrorMessage string
	SectionID    string
	SectionName  string
	StartedAt    time.Time
	Timeout      time.Duration
}

func (s SectionStatus) IsOK() bool {
	return s.ErrorMessage == ""
}

func (s SectionStatus) IsExpired() bool {
	if s.CompletedAt.IsZero() {
		return true
	}
	deadline := s.CompletedAt.Add(s.Timeout)
	return time.Now().After(deadline)
}

func (s SectionStatus) IsCurrent() bool {
	if s.CompletedAt.IsZero() {
		return false
	}
	return time.Now().Before(s.CompletedAt.Add(s.Timeout * 2))
}

func (s SectionStatus) IsMissing() bool {
	return s.CompletedAt.IsZero()
}

func (s SectionStatus) IsRunning() bool {
	return !s.StartedAt.IsZero()
}
