package app

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2/widget"
)

type Status uint

const (
	StatusUnknown Status = iota
	StatusError
	StatusOK
	StatusPartial
	StatusWorking
)

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
		StatusPartial: widget.SuccessImportance,
		StatusUnknown: widget.LowImportance,
		StatusWorking: widget.MediumImportance,
	}
	i, ok := m[s]
	if !ok {
		i = widget.MediumImportance
	}
	return i
}

type StatusSummary struct {
	Current   int
	Errors    int
	IsRunning bool
	Skipped   int
	Missing   int
	Total     int
}

func (ss StatusSummary) ProgressP() float32 {
	return float32(ss.Current+ss.Skipped) / float32(ss.Total)
}

func (ss StatusSummary) Status() Status {
	if ss.Errors > 0 {
		return StatusError
	}
	if ss.ProgressP() == 1 {
		if ss.Skipped > 0 {
			return StatusPartial
		}
		return StatusOK
	}
	return StatusWorking
}

func (ss StatusSummary) DisplayShort() string {
	return ss.display(true)
}

func (ss StatusSummary) Display() string {
	return ss.display(false)
}

func (ss StatusSummary) display(isShort bool) string {
	s := ss.Status()
	if isShort {
		if s == StatusPartial {
			s = StatusOK
		}
	}
	switch s {
	case StatusError:
		return fmt.Sprintf("%d ERRORS", ss.Errors)
	case StatusOK:
		return "OK"
	case StatusPartial:
		return "Partial"
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

type CacheSectionStatus struct {
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

func (ss CacheSectionStatus) IsGeneralSection() bool {
	return ss.EntityID == GeneralSectionEntityID
}

func (ss CacheSectionStatus) HasError() bool {
	return ss.ErrorMessage != ""
}

func (ss CacheSectionStatus) HasComment() bool {
	return ss.Comment != ""
}

func (ss CacheSectionStatus) IsExpired() bool {
	if ss.CompletedAt.IsZero() {
		return true
	}
	deadline := ss.CompletedAt.Add(ss.Timeout)
	return time.Now().After(deadline)
}

func (ss CacheSectionStatus) IsCurrent() bool {
	if ss.CompletedAt.IsZero() {
		return false
	}
	return time.Now().Before(ss.CompletedAt.Add(ss.Timeout * 2))
}

func (ss CacheSectionStatus) IsMissing() bool {
	return ss.CompletedAt.IsZero()
}

func (ss CacheSectionStatus) IsRunning() bool {
	return !ss.StartedAt.IsZero()
}

func (ss CacheSectionStatus) Display() (string, widget.Importance) {
	var s string
	var i widget.Importance
	if ss.HasError() {
		s = "ERROR"
		i = widget.DangerImportance
	} else if ss.IsMissing() {
		s = "Missing"
		i = widget.WarningImportance
	} else if ss.HasComment() {
		s = "Skipped"
		i = widget.MediumImportance
	} else if !ss.IsCurrent() {
		s = "Stale"
		i = widget.WarningImportance
	} else {
		s = "OK"
		i = widget.SuccessImportance
	}
	return s, i
}
