package app

import (
	"fmt"

	"fyne.io/fyne/v2/widget"
)

type Status uint

func (s Status) ToImportance() widget.Importance {
	m := map[Status]widget.Importance{
		StatusError:   widget.DangerImportance,
		StatusMissing: widget.WarningImportance,
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

const (
	StatusUnknown Status = iota
	StatusOK
	StatusError
	StatusMissing
	StatusWorking
)

type StatusSummary struct {
	Current   int
	Errors    int
	Missing   int
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
	if ss.Missing > 0 {
		return StatusMissing
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
	case StatusMissing:
		return fmt.Sprintf("%d Missing", ss.Missing)
	case StatusOK:
		return "OK"
	case StatusWorking:
		return fmt.Sprintf("%.0f%%", ss.ProgressP()*100)
	}
	return "?"
}
