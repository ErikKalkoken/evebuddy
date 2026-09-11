package xwidget_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestNewLabelWithSelection(t *testing.T) {
	l := xwidget.NewLabelWithSelection("Test")
	assert.Equal(t, "Test", l.Text)
	assert.True(t, l.Selectable)
}
