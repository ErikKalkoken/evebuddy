package widget

import "fyne.io/fyne/v2/widget"

func NewRichTextSegmentFromText(s string, inline bool) []widget.RichTextSegment {
	return []widget.RichTextSegment{&widget.TextSegment{Text: s, Style: widget.RichTextStyle{Inline: inline}}}
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
