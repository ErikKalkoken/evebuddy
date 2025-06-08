package widget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/stretchr/testify/assert"
)

func TestNewRichTextSegmentFromText_CanCreateMinimal(t *testing.T) {
	got := iwidget.NewRichTextSegmentFromText("Test")
	x := got[0].(*widget.TextSegment)
	assert.Equal(t, "Test", x.Text)
}

func TestNewRichTextSegmentFromText_CanCreateWithStyle(t *testing.T) {
	got := iwidget.NewRichTextSegmentFromText("Test", widget.RichTextStyle{
		ColorName: theme.ColorNameError,
	})
	x := got[0].(*widget.TextSegment)
	assert.Equal(t, "Test", x.Text)
	assert.Equal(t, theme.ColorNameError, x.Style.ColorName)
}

func TestRichText_CanCreateDefault(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	rt := iwidget.NewRichText(iwidget.NewRichTextSegmentFromText("Test")...)
	w := test.NewWindow(rt)
	defer w.Close()

	test.AssertImageMatches(t, "richtext/default.png", w.Canvas().Capture())
}

func TestRichText_CanCreateWithText(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	rt := iwidget.NewRichTextWithText("Test")
	w := test.NewWindow(rt)
	defer w.Close()

	test.AssertImageMatches(t, "richtext/default.png", w.Canvas().Capture())
}

func TestRichText_CanSet(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	rt := iwidget.NewRichText(iwidget.NewRichTextSegmentFromText("Test")...)
	w := test.NewWindow(rt)
	defer w.Close()

	rt.Set(iwidget.NewRichTextSegmentFromText("XXX"))

	test.AssertImageMatches(t, "richtext/set_text.png", w.Canvas().Capture())
}

func TestRichText_CanSetWithText(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	rt := iwidget.NewRichText(iwidget.NewRichTextSegmentFromText("Test")...)
	w := test.NewWindow(rt)
	defer w.Close()

	rt.SetWithText("XXX")

	test.AssertImageMatches(t, "richtext/set_text.png", w.Canvas().Capture())
}

func TestInlineRichText_CanInline(t *testing.T) {
	a := iwidget.NewRichTextSegmentFromText("a")
	b := iwidget.NewRichTextSegmentFromText("b")
	got := iwidget.InlineRichTextSegments(a, b)
	assert.Len(t, got, 2)
	s1 := got[0].(*widget.TextSegment)
	assert.Equal(t, "a", s1.Text)
	assert.True(t, s1.Style.Inline)
	s2 := got[1].(*widget.TextSegment)
	assert.Equal(t, "b", s2.Text)
	assert.False(t, s2.Style.Inline)
}

func TestInlineRichText_CanInline2(t *testing.T) {
	a := iwidget.NewRichTextSegmentFromText("a", widget.RichTextStyle{Inline: true})
	b := iwidget.NewRichTextSegmentFromText("b")
	got := iwidget.InlineRichTextSegments(a, b)
	assert.Len(t, got, 2)
	s1 := got[0].(*widget.TextSegment)
	assert.Equal(t, "a", s1.Text)
	assert.True(t, s1.Style.Inline)
	s2 := got[1].(*widget.TextSegment)
	assert.Equal(t, "b", s2.Text)
	assert.False(t, s2.Style.Inline)
}

func TestInlineRichText_SkipNonInlinable(t *testing.T) {
	a := iwidget.NewRichTextSegmentFromText("a")
	x := []widget.RichTextSegment{&widget.ImageSegment{Title: "x"}}
	b := iwidget.NewRichTextSegmentFromText("b")
	got := iwidget.InlineRichTextSegments(a, x, b)
	assert.Len(t, got, 2)
	s1 := got[0].(*widget.TextSegment)
	assert.Equal(t, "a", s1.Text)
	assert.True(t, s1.Style.Inline)
	s2 := got[1].(*widget.TextSegment)
	assert.Equal(t, "b", s2.Text)
	assert.False(t, s2.Style.Inline)
}

func TestAlignRichTextSegments_CanAlign(t *testing.T) {
	a := iwidget.NewRichTextSegmentFromText("a")
	b := iwidget.NewRichTextSegmentFromText("b")
	got := iwidget.AlignRichTextSegments(fyne.TextAlignCenter, a, b)
	assert.Len(t, got, 2)
	s1 := got[0].(*widget.TextSegment)
	assert.Equal(t, "a", s1.Text)
	assert.Equal(t, fyne.TextAlignCenter, s1.Style.Alignment)
	s2 := got[1].(*widget.TextSegment)
	assert.Equal(t, "b", s2.Text)
	assert.Equal(t, fyne.TextAlignCenter, s2.Style.Alignment)
}

func TestAlignRichTextSegments_SkipNonAlignable(t *testing.T) {
	a := iwidget.NewRichTextSegmentFromText("a")
	x := []widget.RichTextSegment{&widget.ImageSegment{Title: "x"}}
	b := iwidget.NewRichTextSegmentFromText("b")
	got := iwidget.AlignRichTextSegments(fyne.TextAlignCenter, a, x, b)
	assert.Len(t, got, 2)
	s1 := got[0].(*widget.TextSegment)
	assert.Equal(t, "a", s1.Text)
	assert.Equal(t, fyne.TextAlignCenter, s1.Style.Alignment)
	s2 := got[1].(*widget.TextSegment)
	assert.Equal(t, "b", s2.Text)
	assert.Equal(t, fyne.TextAlignCenter, s2.Style.Alignment)
}
