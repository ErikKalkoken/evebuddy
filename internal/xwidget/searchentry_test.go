package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestSearchEntry_CanCreate(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	se := xwidget.NewSearchEntry("Search...", nil)
	w := test.NewWindow(se)
	defer w.Close()

	assert.Equal(t, "Search...", se.PlaceHolder)
}

func TestSearchEntry_SetTextCallsOnChanged(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var got string
	se := xwidget.NewSearchEntry("Search...", func(s string) {
		got = s
	})
	w := test.NewWindow(se)
	defer w.Close()

	se.SetText("abc")

	assert.Equal(t, "abc", se.Text)
	assert.Equal(t, "abc", got)
}

func TestSearchEntry_ClearSilentResetsTextWithoutCallback(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var callCount int
	se := xwidget.NewSearchEntry("Search...", func(s string) {
		callCount++
	})
	w := test.NewWindow(se)
	defer w.Close()

	se.SetText("abc")
	assert.Equal(t, 1, callCount)

	se.ClearSilent()
	assert.Equal(t, "", se.Text)
	assert.Equal(t, 1, callCount)
}
