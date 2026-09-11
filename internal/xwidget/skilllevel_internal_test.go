package xwidget

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
)

func TestSkillLevel_Set_ChoosesCorrectIconPerDot(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	sl := NewSkillLevel()

	t.Run("all untrained by default", func(t *testing.T) {
		sl.Set(0, 0, 0)
		for i := range 5 {
			assert.Equal(t, sl.disabled, sl.dots[i].Resource)
		}
	})
	t.Run("active levels show trained icon, blocked levels show blocked icon, queued shows queued icon", func(t *testing.T) {
		sl.Set(2, 4, 5)
		assert.Equal(t, sl.trained, sl.dots[0].Resource)
		assert.Equal(t, sl.trained, sl.dots[1].Resource)
		assert.Equal(t, sl.blocked, sl.dots[2].Resource)
		assert.Equal(t, sl.blocked, sl.dots[3].Resource)
		assert.Equal(t, sl.queued, sl.dots[4].Resource)
	})
	t.Run("queued is ignored when zero", func(t *testing.T) {
		sl.Set(1, 3, 0)
		assert.Equal(t, sl.trained, sl.dots[0].Resource)
		assert.Equal(t, sl.blocked, sl.dots[1].Resource)
		assert.Equal(t, sl.blocked, sl.dots[2].Resource)
		assert.Equal(t, sl.disabled, sl.dots[3].Resource)
		assert.Equal(t, sl.disabled, sl.dots[4].Resource)
	})
}
