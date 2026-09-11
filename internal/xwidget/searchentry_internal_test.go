package xwidget

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
)

func TestSearchEntry_ClearButtonTogglesWithText(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	se := NewSearchEntry("Search...", nil)
	w := test.NewWindow(se)
	defer w.Close()

	assert.False(t, se.clearButton.Visible())

	se.SetText("abc")
	assert.True(t, se.clearButton.Visible())

	se.SetText("")
	assert.False(t, se.clearButton.Visible())
}

func TestSearchEntry_ClearButtonTapClearsText(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var got string
	se := NewSearchEntry("Search...", func(s string) {
		got = s
	})
	w := test.NewWindow(se)
	defer w.Close()

	se.SetText("abc")
	test.Tap(se.clearButton)

	assert.Equal(t, "", se.Text)
	assert.Equal(t, "", got)
}
