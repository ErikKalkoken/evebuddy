package app

import (
	"fmt"

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
