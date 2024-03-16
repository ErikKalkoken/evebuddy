package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2/data/binding"
)

type statusLabel struct {
	label binding.String
}

func (s statusLabel) SetText(format string, a ...any) {
	err := s.label.Set(fmt.Sprintf(format, a...))
	if err != nil {
		log.Printf("Failed to set status label: %v", err)
	}
}

func (s statusLabel) Clear() {
	s.SetText("")
}

func newStatusLabel(label binding.String) statusLabel {
	o := statusLabel{label: label}
	return o
}
