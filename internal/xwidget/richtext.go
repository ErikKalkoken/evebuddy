package xwidget

import (
	"slices"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// RichTextSegmentsFromText returns a text segment from s.
// The optional style is applied to the segment.
func RichTextSegmentsFromText(s string, style ...widget.RichTextStyle) []widget.RichTextSegment {
	seg := &widget.TextSegment{Text: s}
	if len(style) > 0 {
		seg.Style = style[0]
	}
	return []widget.RichTextSegment{seg}
}

// RichTextSegmentsInlined returns an inlined copy of the text segments,
// so they are all rendered in the same line.
// Non-text segments are skipped.
func RichTextSegmentsInlined(segments ...[]widget.RichTextSegment) []widget.RichTextSegment {
	var texts []*widget.TextSegment
	for _, s := range slices.Concat(segments...) {
		if t, ok := s.(*widget.TextSegment); ok {
			texts = append(texts, t)
		}
	}
	segs2 := make([]widget.RichTextSegment, 0, len(texts))
	for i, t := range texts {
		t2 := copyTextSegment(t)
		if i < len(texts)-1 {
			t2.Style.Inline = true
		}
		segs2 = append(segs2, t2)
	}
	return segs2
}

// RichTextSegmentsWithStyle returns a copy where the style of all text segments is modified by f.
// Non-text segments are kept unchanged.
func RichTextSegmentsWithStyle(segments []widget.RichTextSegment, f func(x *widget.RichTextStyle)) []widget.RichTextSegment {
	segs2 := make([]widget.RichTextSegment, 0, len(segments))
	for _, x := range segments {
		t, ok := x.(*widget.TextSegment)
		if !ok {
			segs2 = append(segs2, x)
			continue
		}
		t2 := copyTextSegment(t)
		f(&t2.Style)
		segs2 = append(segs2, t2)
	}
	return segs2
}

// copyTextSegment returns a new text segment with the text and style of t.
func copyTextSegment(t *widget.TextSegment) *widget.TextSegment {
	return &widget.TextSegment{Text: t.Text, Style: t.Style}
}

// RichText is a variant of [widget.RichText] which adds convenience setters.
type RichText struct {
	widget.RichText
}

// NewRichText returns a new [RichText] with the given segments.
func NewRichText(segments ...widget.RichTextSegment) *RichText {
	w := &RichText{}
	w.ExtendBaseWidget(w)
	w.Segments = segments
	w.Scroll = container.ScrollNone
	return w
}

// NewRichTextWithText returns a new [RichText] showing text.
// The optional style is applied to the text.
func NewRichTextWithText(text string, style ...widget.RichTextStyle) *RichText {
	return NewRichText(RichTextSegmentsFromText(text, style...)...)
}

// Set replaces the segments and refreshes the widget.
func (w *RichText) Set(segments []widget.RichTextSegment) {
	w.Segments = segments
	w.Refresh()
}

// SetWithText replaces the segments with text and refreshes the widget.
// The optional style is applied to the text.
func (w *RichText) SetWithText(text string, style ...widget.RichTextStyle) {
	w.Set(RichTextSegmentsFromText(text, style...))
}
