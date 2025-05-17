package widget

import (
	"slices"

	"fyne.io/fyne/v2"
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

// InlineRichTextSegments returns an inlined copy of the segments,
// so they are all rendered in the same line.
func InlineRichTextSegments(segs ...[]widget.RichTextSegment) []widget.RichTextSegment {
	seg := slices.Concat(segs...)
	for _, x := range seg[:len(seg)-1] {
		t, ok := x.(*widget.TextSegment)
		if !ok {
			continue
		}
		t.Style.Inline = true
	}
	return seg
}

// AlignRichTextSegments returns a copy where all segmens are aligned as given.
func AlignRichTextSegments(align fyne.TextAlign, segs ...[]widget.RichTextSegment) []widget.RichTextSegment {
	seg := slices.Concat(segs...)
	for _, x := range seg {
		t, ok := x.(*widget.TextSegment)
		if !ok {
			continue
		}
		t.Style.Alignment = align
	}
	return seg
}
