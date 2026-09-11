package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestNewSpacer_HasGivenMinSize(t *testing.T) {
	s := xwidget.NewSpacer(fyne.NewSize(10, 20))
	assert.Equal(t, fyne.NewSize(10, 20), s.MinSize())
}

func TestNewStandardSpacer_HasPaddingSize(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	s := xwidget.NewStandardSpacer()
	p := theme.Padding()
	assert.Equal(t, fyne.NewSquareSize(p), s.MinSize())
}
