package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestSkillLevel_CanCreateAndSet(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	sl := xwidget.NewSkillLevel()
	w := test.NewWindow(sl)
	defer w.Close()

	assert.NotPanics(t, func() {
		sl.Set(2, 4, 5)
	})
}
