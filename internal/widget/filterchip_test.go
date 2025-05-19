package widget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/stretchr/testify/assert"
)

func TestFilterChip(t *testing.T) {
	test.NewTempApp(t)
	t.Run("can set state", func(t *testing.T) {
		x := iwidget.NewFilterChip("dummy", nil)
		x.SetState(true)
		assert.True(t, x.On)
		x.SetState(false)
		assert.False(t, x.On)
	})
	t.Run("cb is called on change", func(t *testing.T) {
		var isCalled, v bool
		x := iwidget.NewFilterChip("dummy", func(on bool) {
			isCalled = true
			v = on
		})
		x.SetState(true)
		assert.True(t, isCalled)
		assert.True(t, v)
	})
}
