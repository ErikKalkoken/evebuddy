package xwidget

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
)

func TestFilterChipCompact_Menu(t *testing.T) {
	test.NewTempApp(t)
	t.Run("should create initial menu", func(t *testing.T) {
		// when
		f := NewFilterChipCompact([]FilterOption{
			NewFilterOptionToogle("Alpha"),
		}, nil)

		// then
		assert.Len(t, f.menu.Items, 3)
		it := f.menu.Items[0]
		assert.Equal(t, "Alpha", it.Label)
		assert.Equal(t, f.blankResource, it.Icon)
	})
}
