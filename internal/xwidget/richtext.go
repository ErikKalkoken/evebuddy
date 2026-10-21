package xwidget

import (
	"log/slog"
	"slices"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

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

type RichText struct {
	widget.RichText
}

func NewRichText(segments ...widget.RichTextSegment) *RichText {
	w := &RichText{}
	w.ExtendBaseWidget(w)
	w.Segments = segments
	w.Scroll = container.ScrollNone
	return w
}

func NewRichTextWithText(text string, style ...widget.RichTextStyle) *RichText {
	return NewRichText(RichTextSegmentsFromText(text, style...)...)
}

func (w *RichText) Set(segments []widget.RichTextSegment) {
	w.Segments = segments
	w.Refresh()
}

func (w *RichText) SetWithText(text string, style ...widget.RichTextStyle) {
	w.Set(RichTextSegmentsFromText(text, style...))
}
