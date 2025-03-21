package widget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/stretchr/testify/assert"
)

// TODO: Add more tests

func TestSnackbar(t *testing.T) {
	w := test.NewWindow(widget.NewLabel(""))
	defer w.Close()
	sb := iwidget.NewSnackbar(w)
	sb.Start()
	assert.True(t, sb.IsRunning())
	sb.Show("Dummy")
}
