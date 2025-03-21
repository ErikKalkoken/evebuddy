// Package widgets contains generic Fyne widgets.
package widget

import (
	"time"

	"fyne.io/fyne/v2/widget"
)

const (
	defaultAnimationDuration = 300 * time.Millisecond
)

// SetRichText sets the content of a RichtText widget and refreshes it.
func SetRichText(w *widget.RichText, ts ...widget.TextSegment) {
	segs := make([]widget.RichTextSegment, len(ts))
	for i, s := range ts {
		segs[i] = &s
	}
	w.Segments = segs
	w.Refresh()
}
