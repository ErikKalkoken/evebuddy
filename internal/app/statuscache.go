package app

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2/widget"
)

type Status uint

func (s Status) ToImportance() widget.Importance {
	m := map[Status]widget.Importance{
		StatusError:   widget.DangerImportance,
		StatusOK:      widget.MediumImportance,
		StatusUnknown: widget.LowImportance,
		StatusWorking: widget.MediumImportance,
	}
	i, ok := m[s]
	if !ok {
		i = widget.MediumImportance
	}
	return i
}

func (s Status) ToImportance2() widget.Importance {
	m := map[Status]widget.Importance{
		StatusError:   widget.DangerImportance,
		StatusOK:      widget.SuccessImportance,
		StatusUnknown: widget.LowImportance,
		StatusWorking: widget.MediumImportance,
	}
	i, ok := m[s]
	if !ok {
		i = widget.MediumImportance
	}
	return i
}

const (
	StatusUnknown Status = iota
	StatusOK
	StatusError
	StatusWorking
)

type StatusSummary struct {
	Current   int
	Errors    int
	IsRunning bool
	Total     int
}

func (ss StatusSummary) ProgressP() float32 {
	return float32(ss.Current) / float32(ss.Total)
}

func (ss StatusSummary) Status() Status {
	if ss.Errors > 0 {
		return StatusError
	}
	if ss.ProgressP() == 1 {
		return StatusOK
	}
	return StatusWorking
}

func (ss StatusSummary) Display() string {
	switch ss.Status() {
	case StatusError:
		return fmt.Sprintf("%d ERRORS", ss.Errors)
	case StatusOK:
		return "OK"
	case StatusWorking:
		return fmt.Sprintf("%.0f%%", ss.ProgressP()*100)
	}
	return "?"
}

// Entity ID for general sections
const (
	GeneralSectionEntityID   = 1
	GeneralSectionEntityName = "Eve Universe"
)

type SectionStatus struct {
	EntityID     int32
	EntityName   string
	Comment      string
	CompletedAt  time.Time
	ContentHash  string
	ErrorMessage string
	SectionID    string
	SectionName  string
	StartedAt    time.Time
	Timeout      time.Duration
}

func (s SectionStatus) IsGeneralSection() bool {
	return s.EntityID == GeneralSectionEntityID
}

func (s SectionStatus) HasError() bool {
	return s.ErrorMessage != ""
}

func (s SectionStatus) HasComment() bool {
	return s.Comment != ""
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
