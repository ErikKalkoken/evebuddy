package xwidget

import (
	"log/slog"
	"slices"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// RichTextSegmentsFromText returns a single text segment for s.
// The optional style is applied to the segment.
func RichTextSegmentsFromText(s string, style ...widget.RichTextStyle) []widget.RichTextSegment {
	seg := &widget.TextSegment{Text: s}
	if len(style) > 0 {
		seg.Style = style[0]
	}
	return []widget.RichTextSegment{seg}
}

// RichTextSegmentsInlined returns an inlined copy of the segments,
// so they are all rendered in the same line.
// Non-inlinable segments are skipped.
func RichTextSegmentsInlined(segments ...[]widget.RichTextSegment) []widget.RichTextSegment {
	segs := slices.Concat(segments...)
	if len(segs) == 0 {
		return nil
	}
	var segs2 []widget.RichTextSegment
	for _, s := range segs[:len(segs)-1] {
		if s.Inline() {
			segs2 = append(segs2, s)
			continue
		}
		t, ok := s.(*widget.TextSegment)
		if !ok {
			slog.Warn("RichTextSegmentsInlined: Skipping non inlinable segment")
			continue
		}
		t2 := *t
		t2.Style.Inline = true
		segs2 = append(segs2, &t2)
	}
	segs2 = append(segs2, segs[len(segs)-1])
	return segs2
}

// RichTextSegmentsWithStyle returns a copy where the style of all segments is modified by f.
// Non-text segments are skipped.
func RichTextSegmentsWithStyle(segments []widget.RichTextSegment, f func(x *widget.RichTextStyle)) []widget.RichTextSegment {
	var segs2 []widget.RichTextSegment
	for _, x := range segments {
		t, ok := x.(*widget.TextSegment)
		if !ok {
			continue
		}
		t2 := *t
		f(&t2.Style)
		segs2 = append(segs2, &t2)
	}
	return segs2
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
