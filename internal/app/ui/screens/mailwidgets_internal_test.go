package screens

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func TestMailHeaderWidget_Owner(t *testing.T) {
	test.NewTempApp(t)
	newWidget := func() *MailHeaderWidget {
		w := NewMailHeaderWidget(func(_ *app.EveEntity, _ int, _ func(fyne.Resource)) {}, func(*app.EveEntity) {})
		w.Set(&app.EveEntity{ID: 1, Name: "Sender"}, time.Now())
		return w
	}
	t.Run("hides owner by default", func(t *testing.T) {
		w := newWidget()
		assert.False(t, w.ownerRow.Visible())
	})
	t.Run("shows owner", func(t *testing.T) {
		w := newWidget()
		w.SetOwner("Bruce Wayne")
		assert.Equal(t, "[Bruce Wayne]", w.owner.Text)
		assert.True(t, w.ownerRow.Visible())
	})
	t.Run("empty name hides owner", func(t *testing.T) {
		w := newWidget()
		w.SetOwner("Bruce Wayne")
		w.SetOwner("")
		assert.False(t, w.ownerRow.Visible())
	})
	t.Run("clear hides owner", func(t *testing.T) {
		w := newWidget()
		w.SetOwner("Bruce Wayne")
		w.Clear()
		assert.False(t, w.ownerRow.Visible())
	})
}
