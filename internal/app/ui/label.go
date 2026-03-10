package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// NewLabelWithTruncation returns a new label with default truncation.
func NewLabelWithTruncation(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Truncation = fyne.TextTruncateEllipsis
	return l
}

// NewLabelWithWrapping returns a new label with default wrapping.
func NewLabelWithWrapping(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Wrapping = fyne.TextWrapWord
	return l
}

// TODO: Remove this helper

// MakeTopText makes the content for the top label of a gui element.
func MakeTopText(characterID int64, hasData bool, err error, create func() (string, widget.Importance)) (string, widget.Importance) {
	if err != nil {
		return "ERROR: " + app.ErrorDisplay(err), widget.DangerImportance
	}
	if characterID == 0 {
		return "No entity", widget.LowImportance
	}
	if !hasData {
		return "No data", widget.WarningImportance
	}
	if create == nil {
		return "", widget.MediumImportance
	}
	return create()
}
