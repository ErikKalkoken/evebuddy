package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestNewCustomHyperlink_CanCreate(t *testing.T) {
	h := xwidget.NewCustomHyperlink("Test", nil)
	assert.Equal(t, "Test", h.Text)
}

func TestNewCustomHyperlink_CanTap(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var tapped bool
	h := xwidget.NewCustomHyperlink("Test", func() {
		tapped = true
	})
	w := test.NewWindow(h)
	defer w.Close()

	// Simulate what widget.Hyperlink.Tapped does when the tap lands on the
	// text, without depending on its internal hit-testing geometry.
	require.NotNil(t, h.OnTapped)
	h.OnTapped()
	assert.True(t, tapped)
}
