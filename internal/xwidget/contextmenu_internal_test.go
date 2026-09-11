package xwidget

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
)

func TestContextMenuButton_SetMenuItems(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	b := NewContextMenuButton("Test", fyne.NewMenu("", fyne.NewMenuItem("Old", nil)))
	newItems := []*fyne.MenuItem{fyne.NewMenuItem("New", nil)}

	b.SetMenuItems(newItems)

	assert.Equal(t, newItems, b.menu.Items)
}
