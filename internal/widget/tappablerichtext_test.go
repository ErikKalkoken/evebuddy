package widget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

func TestTappableRichText_CanCreate(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	text := iwidget.NewTappableRichText(nil, iwidget.NewRichTextSegmentFromText("Test")...)
	w := test.NewWindow(text)
	defer w.Close()

	test.AssertImageMatches(t, "tappablerichtext/default.png", w.Canvas().Capture())
}

func TestTappableRichText_CanCreateWithText(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	text := iwidget.NewTappableRichTextWithText("Test", nil)
	w := test.NewWindow(text)
	defer w.Close()

	test.AssertImageMatches(t, "tappablerichtext/default.png", w.Canvas().Capture())
}

func TestTappableRichText_CanTap(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	var tapped bool
	text := iwidget.NewTappableRichText(func() {
		tapped = true
		test.ApplyTheme(t, test.Theme())
	})
	w := test.NewWindow(text)
	defer w.Close()

	test.Tap(text)
	assert.True(t, tapped)
}

func TestTappableRichText_IgnoreTapWhenNoCallback(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())
	text := iwidget.NewTappableRichText(nil, iwidget.NewRichTextSegmentFromText("Test")...)
	w := test.NewWindow(text)
	defer w.Close()

	test.Tap(text)
}
