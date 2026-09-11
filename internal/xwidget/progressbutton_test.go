package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestProgressButton_CanCreate(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	pb := xwidget.NewProgressButton("Click", theme.HomeIcon(), nil)
	w := test.NewWindow(pb)
	defer w.Close()

	assert.False(t, pb.Disabled())
}

func TestProgressButton_EnableDisable(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	pb := xwidget.NewProgressButton("Click", theme.HomeIcon(), nil)
	w := test.NewWindow(pb)
	defer w.Close()

	pb.Disable()
	assert.True(t, pb.Disabled())

	pb.Enable()
	assert.False(t, pb.Disabled())
}
