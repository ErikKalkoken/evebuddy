package widget

import (
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func NewRichTextSegmentFromText(s string, style ...widget.RichTextStyle) []widget.RichTextSegment {
	seg := &widget.TextSegment{Text: s}
	if len(style) > 0 {
		seg.Style = style[0]
	}
	return []widget.RichTextSegment{seg}
}

// InlineRichTextSegments returns an inlined copy of the segments,
// so they are all rendered in the same line.
// Non-alignable segments are skipped.
func InlineRichTextSegments(segments ...[]widget.RichTextSegment) []widget.RichTextSegment {
	segs2 := make([]widget.RichTextSegment, 0)
	segs := slices.Concat(segments...)
	for _, s := range segs[:len(segs)-1] {
		if s.Inline() {
			segs2 = append(segs2, s)
			continue
		}
		t, ok := s.(*widget.TextSegment)
		if !ok {
			slog.Warn("Skipping non inlinable segment")
			continue
		}
		t.Style.Inline = true
		segs2 = append(segs2, t)
	}
	segs2 = append(segs2, segs[len(segs)-1])
	return segs2
}

// AlignRichTextSegments returns a copy where all segments are aligned as given.
// Non-alignable segments are skipped.
func AlignRichTextSegments(align fyne.TextAlign, segments ...[]widget.RichTextSegment) []widget.RichTextSegment {
	segs2 := make([]widget.RichTextSegment, 0)
	segs := slices.Concat(segments...)
	for _, x := range segs {
		t, ok := x.(*widget.TextSegment)
		if !ok {
			continue
		}
		t.Style.Alignment = align
		segs2 = append(segs2, t)
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

func NewRichTextWithText(text string) *RichText {
	return NewRichText(NewRichTextSegmentFromText(text)...)
}

func (w *RichText) Set(segments []widget.RichTextSegment) {
	w.Segments = segments
	w.Refresh()
}

func (w *RichText) SetWithText(text string) {
	w.Set(NewRichTextSegmentFromText(text))
}
