package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestNewRichTextSegmentFromText_CanCreateMinimal(t *testing.T) {
	got := xwidget.RichTextSegmentsFromText("Test")
	x := got[0].(*widget.TextSegment)
	assert.Equal(t, "Test", x.Text)
}

func TestNewRichTextSegmentFromText_CanCreateWithStyle(t *testing.T) {
	got := xwidget.RichTextSegmentsFromText("Test", widget.RichTextStyle{
		ColorName: theme.ColorNameError,
	})
	x := got[0].(*widget.TextSegment)
	assert.Equal(t, "Test", x.Text)
	assert.Equal(t, theme.ColorNameError, x.Style.ColorName)
}

func TestRichText_CanCreateDefault(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	rt := xwidget.NewRichText(xwidget.RichTextSegmentsFromText("Test")...)
	w := test.NewWindow(rt)
	defer w.Close()

	test.AssertImageMatches(t, "richtext/default.png", w.Canvas().Capture())
}

func TestRichText_CanCreateWithText(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	rt := xwidget.NewRichTextWithText("Test")
	w := test.NewWindow(rt)
	defer w.Close()

	test.AssertImageMatches(t, "richtext/default.png", w.Canvas().Capture())
}

func TestRichText_CanSet(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	rt := xwidget.NewRichText(xwidget.RichTextSegmentsFromText("Test")...)
	w := test.NewWindow(rt)
	defer w.Close()

	rt.Set(xwidget.RichTextSegmentsFromText("XXX"))

	test.AssertImageMatches(t, "richtext/set_text.png", w.Canvas().Capture())
}

func TestRichText_CanSetWithText(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	rt := xwidget.NewRichText(xwidget.RichTextSegmentsFromText("Test")...)
	w := test.NewWindow(rt)
	defer w.Close()

	rt.SetWithText("XXX")

	test.AssertImageMatches(t, "richtext/set_text.png", w.Canvas().Capture())
}

func TestRichTextSegmentsInlined_CanInline(t *testing.T) {
	a := xwidget.RichTextSegmentsFromText("a")
	b := xwidget.RichTextSegmentsFromText("b")
	got := xwidget.RichTextSegmentsInlined(a, b)
	assert.Len(t, got, 2)
	s1 := got[0].(*widget.TextSegment)
	assert.Equal(t, "a", s1.Text)
	assert.True(t, s1.Style.Inline)
	s2 := got[1].(*widget.TextSegment)
	assert.Equal(t, "b", s2.Text)
	assert.False(t, s2.Style.Inline)
}

func TestRichTextSegmentsInlined_CanInline2(t *testing.T) {
	a := xwidget.RichTextSegmentsFromText("a", widget.RichTextStyle{Inline: true})
	b := xwidget.RichTextSegmentsFromText("b")
	got := xwidget.RichTextSegmentsInlined(a, b)
	assert.Len(t, got, 2)
	s1 := got[0].(*widget.TextSegment)
	assert.Equal(t, "a", s1.Text)
	assert.True(t, s1.Style.Inline)
	s2 := got[1].(*widget.TextSegment)
	assert.Equal(t, "b", s2.Text)
	assert.False(t, s2.Style.Inline)
}

func TestRichTextSegmentsInlined_SkipNonInlinable(t *testing.T) {
	a := xwidget.RichTextSegmentsFromText("a")
	x := []widget.RichTextSegment{&widget.ImageSegment{Title: "x"}}
	b := xwidget.RichTextSegmentsFromText("b")
	got := xwidget.RichTextSegmentsInlined(a, x, b)
	assert.Len(t, got, 2)
	s1 := got[0].(*widget.TextSegment)
	assert.Equal(t, "a", s1.Text)
	assert.True(t, s1.Style.Inline)
	s2 := got[1].(*widget.TextSegment)
	assert.Equal(t, "b", s2.Text)
	assert.False(t, s2.Style.Inline)
}

func TestRichTextSegmentsInlined_DoesNotModifyInput(t *testing.T) {
	// given
	a := xwidget.RichTextSegmentsFromText("a")
	b := xwidget.RichTextSegmentsFromText("b")
	// when
	xwidget.RichTextSegmentsInlined(a, b)
	// then
	assert.False(t, a[0].(*widget.TextSegment).Style.Inline)
}

func TestRichTextSegmentsInlined_ReturnsCopies(t *testing.T) {
	// given
	a := xwidget.RichTextSegmentsFromText("a", widget.RichTextStyle{Inline: true})
	b := xwidget.RichTextSegmentsFromText("b")
	// when
	got := xwidget.RichTextSegmentsInlined(a, b)
	// then
	assert.Len(t, got, 2)
	assert.NotSame(t, a[0], got[0])
	assert.Equal(t, "a", got[0].(*widget.TextSegment).Text)
	assert.True(t, got[0].(*widget.TextSegment).Style.Inline)
	assert.NotSame(t, b[0], got[1])
	assert.Equal(t, "b", got[1].(*widget.TextSegment).Text)
	assert.False(t, got[1].(*widget.TextSegment).Style.Inline)
}

func TestRichTextSegmentsInlined_SkipNonTextLastSegment(t *testing.T) {
	// given
	a := xwidget.RichTextSegmentsFromText("a")
	x := []widget.RichTextSegment{&widget.ImageSegment{Title: "x"}}
	// when
	got := xwidget.RichTextSegmentsInlined(a, x)
	// then
	assert.Len(t, got, 1)
	assert.Equal(t, "a", got[0].(*widget.TextSegment).Text)
}

func TestRichTextSegmentsInlined_ReturnsEmptyForEmptyInput(t *testing.T) {
	// when
	got := xwidget.RichTextSegmentsInlined()
	// then
	assert.Empty(t, got)
}

func TestRichTextSegmentsWithStyle_CanModify(t *testing.T) {
	// given
	a := xwidget.RichTextSegmentsFromText("a")
	// when
	got := xwidget.RichTextSegmentsWithStyle(a, func(x *widget.RichTextStyle) {
		x.ColorName = theme.ColorNameError
	})
	// then
	assert.Len(t, got, 1)
	s1 := got[0].(*widget.TextSegment)
	assert.Equal(t, "a", s1.Text)
	assert.Equal(t, theme.ColorNameError, s1.Style.ColorName)
}

func TestRichTextSegmentsWithStyle_KeepsNonTextUnchanged(t *testing.T) {
	// given
	x := &widget.ImageSegment{Title: "x"}
	segs := []widget.RichTextSegment{x, &widget.TextSegment{Text: "a"}}
	// when
	got := xwidget.RichTextSegmentsWithStyle(segs, func(s *widget.RichTextStyle) {
		s.ColorName = theme.ColorNameError
	})
	// then
	assert.Len(t, got, 2)
	assert.Same(t, x, got[0])
	s2 := got[1].(*widget.TextSegment)
	assert.Equal(t, "a", s2.Text)
	assert.Equal(t, theme.ColorNameError, s2.Style.ColorName)
}

func TestRichTextSegmentsWithStyle_DoesNotModifyInput(t *testing.T) {
	// given
	a := xwidget.RichTextSegmentsFromText("a")
	// when
	xwidget.RichTextSegmentsWithStyle(a, func(x *widget.RichTextStyle) {
		x.ColorName = theme.ColorNameError
	})
	// then
	assert.Equal(t, fyne.ThemeColorName(""), a[0].(*widget.TextSegment).Style.ColorName)
}
