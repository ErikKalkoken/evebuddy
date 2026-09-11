package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestActivity_CanCreate(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := xwidget.NewActivity()
	w := test.NewWindow(a)
	defer w.Close()

	assert.NotPanics(t, a.Start)
	assert.NotPanics(t, a.Stop)
}

func TestActivity_SupportsToolTip(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := xwidget.NewActivity()
	w := test.NewWindow(a)
	defer w.Close()

	a.SetToolTip("Loading")
	assert.Equal(t, "Loading", a.ToolTip())
}
