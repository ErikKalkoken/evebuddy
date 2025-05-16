package widget

import (
	"slices"

	"fyne.io/fyne/v2/widget"
)

func NewRichTextSegmentFromText(s string, style ...widget.RichTextStyle) []widget.RichTextSegment {
	seg := &widget.TextSegment{Text: s}
	if len(style) > 0 {
		seg.Style = style[0]
	}
	return []widget.RichTextSegment{seg}
}

// SetRichText sets the content of a RichtText widget and refreshes it.
func SetRichText(w *widget.RichText, segs ...widget.RichTextSegment) {
	w.Segments = segs
	w.Refresh()
}

// InlineRichTextSegments returns an inlined copy of the segments, so they are rendered in the same line.
func InlineRichTextSegments(s []widget.RichTextSegment) []widget.RichTextSegment {
	s2 := slices.Clone(s)
	for _, x := range s2[:len(s2)-1] {
		t, ok := x.(*widget.TextSegment)
		if !ok {
			continue
		}
		t.Style.Inline = true
	}
	return s
}
