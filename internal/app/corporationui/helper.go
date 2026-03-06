package corporationui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type loadFuncAsync func(int64, int, func(fyne.Resource))

func newLabelWithTruncation() *widget.Label {
	l := widget.NewLabel("")
	l.Truncation = fyne.TextTruncateEllipsis
	return l
}

// corporationIDOrZero returns the ID of a corporation or 0 if the c does not exist.
func corporationIDOrZero(c *app.Corporation) int64 {
	if c == nil {
		return 0
	}
	return c.ID
}

// makeTopText makes the content for the top label of a gui element.
func makeTopText(characterID int64, hasData bool, err error, make func() (string, widget.Importance)) (string, widget.Importance) {
	if err != nil {
		return "ERROR: " + app.ErrorDisplay(err), widget.DangerImportance
	}
	if characterID == 0 {
		return "No entity", widget.LowImportance
	}
	if !hasData {
		return "No data", widget.WarningImportance
	}
	if make == nil {
		return "", widget.MediumImportance
	}
	return make()
}
