package widget

import "fyne.io/fyne/v2/widget"

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

func RichTextSegmentsSetInline(s []widget.RichTextSegment) []widget.RichTextSegment {
	for _, x := range s {
		t, ok := x.(*widget.TextSegment)
		if !ok {
			continue
		}
		t.Style.Inline = true
	}
	return s
}
