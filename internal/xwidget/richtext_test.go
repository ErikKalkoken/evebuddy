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

func TestInlineRichText_CanInline(t *testing.T) {
	a := xwidget.RichTextSegmentsFromText("a")
	b := xwidget.RichTextSegmentsFromText("b")
	got := xwidget.InlineRichTextSegments(a, b)
	assert.Len(t, got, 2)
	s1 := got[0].(*widget.TextSegment)
	assert.Equal(t, "a", s1.Text)
	assert.True(t, s1.Style.Inline)
	s2 := got[1].(*widget.TextSegment)
	assert.Equal(t, "b", s2.Text)
	assert.False(t, s2.Style.Inline)
}

func TestInlineRichText_CanInline2(t *testing.T) {
	a := xwidget.RichTextSegmentsFromText("a", widget.RichTextStyle{Inline: true})
	b := xwidget.RichTextSegmentsFromText("b")
	got := xwidget.InlineRichTextSegments(a, b)
	assert.Len(t, got, 2)
	s1 := got[0].(*widget.TextSegment)
	assert.Equal(t, "a", s1.Text)
	assert.True(t, s1.Style.Inline)
	s2 := got[1].(*widget.TextSegment)
	assert.Equal(t, "b", s2.Text)
	assert.False(t, s2.Style.Inline)
}

func TestInlineRichText_SkipNonInlinable(t *testing.T) {
	a := xwidget.RichTextSegmentsFromText("a")
	x := []widget.RichTextSegment{&widget.ImageSegment{Title: "x"}}
	b := xwidget.RichTextSegmentsFromText("b")
	got := xwidget.InlineRichTextSegments(a, x, b)
	assert.Len(t, got, 2)
	s1 := got[0].(*widget.TextSegment)
	assert.Equal(t, "a", s1.Text)
	assert.True(t, s1.Style.Inline)
	s2 := got[1].(*widget.TextSegment)
	assert.Equal(t, "b", s2.Text)
	assert.False(t, s2.Style.Inline)
}

func TestInlineRichText_DoesNotModifyInput(t *testing.T) {
	// given
	a := xwidget.RichTextSegmentsFromText("a")
	b := xwidget.RichTextSegmentsFromText("b")
	// when
	xwidget.InlineRichTextSegments(a, b)
	// then
	assert.False(t, a[0].(*widget.TextSegment).Style.Inline)
}

func TestInlineRichText_ReturnsNilForEmptyInput(t *testing.T) {
	// when
	got := xwidget.InlineRichTextSegments()
	// then
	assert.Nil(t, got)
}

func TestModifyRichTextStyle_CanModify(t *testing.T) {
	// given
	a := xwidget.RichTextSegmentsFromText("a")
	// when
	got := xwidget.ModifyRichTextStyle(a, func(x *widget.RichTextStyle) {
		x.ColorName = theme.ColorNameError
	})
	// then
	assert.Len(t, got, 1)
	s1 := got[0].(*widget.TextSegment)
	assert.Equal(t, "a", s1.Text)
	assert.Equal(t, theme.ColorNameError, s1.Style.ColorName)
}

func TestModifyRichTextStyle_SkipNonText(t *testing.T) {
	// given
	segs := []widget.RichTextSegment{
		&widget.ImageSegment{Title: "x"},
		&widget.TextSegment{Text: "a"},
	}
	// when
	got := xwidget.ModifyRichTextStyle(segs, func(x *widget.RichTextStyle) {
		x.ColorName = theme.ColorNameError
	})
	// then
	assert.Len(t, got, 1)
	assert.Equal(t, "a", got[0].(*widget.TextSegment).Text)
}

func TestModifyRichTextStyle_DoesNotModifyInput(t *testing.T) {
	// given
	a := xwidget.RichTextSegmentsFromText("a")
	// when
	xwidget.ModifyRichTextStyle(a, func(x *widget.RichTextStyle) {
		x.ColorName = theme.ColorNameError
	})
	// then
	assert.Equal(t, fyne.ThemeColorName(""), a[0].(*widget.TextSegment).Style.ColorName)
}
