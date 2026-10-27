package ui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestInfoLink(t *testing.T) {
	test.NewTempApp(t)

	// parts returns the link and the placeholder label of w.
	parts := func(w *ui.InfoLink) (*widget.Hyperlink, *widget.Label) {
		stack := test.WidgetRenderer(w).Objects()[0].(*fyne.Container)
		return stack.Objects[0].(*widget.Hyperlink), stack.Objects[1].(*widget.Label)
	}

	t.Run("shows placeholder initially", func(t *testing.T) {
		// when
		w := ui.NewInfoLink(&testdouble.InfoViewerFake{})
		// then
		link, label := parts(w)
		assert.False(t, link.Visible())
		assert.True(t, label.Visible())
		xassert.Equal(t, "-", label.Text)
	})
	t.Run("shows link for entity", func(t *testing.T) {
		// given
		iv := &testdouble.InfoViewerFake{}
		w := ui.NewInfoLink(iv)
		ee := &app.EveEntity{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter}
		w.Set(ee)
		link, label := parts(w)
		// when
		test.Tap(link)
		// then
		assert.True(t, link.Visible())
		assert.False(t, label.Visible())
		xassert.Equal(t, "Bruce Wayne", link.Text)
		xassert.Equal(t, []*app.EveEntity{ee}, iv.Shown)
	})
	t.Run("shows link for bloodline", func(t *testing.T) {
		// given
		iv := &testdouble.InfoViewerFake{}
		w := ui.NewInfoLink(iv)
		w.SetBloodline(&app.EntityShort{ID: 7, Name: "Deteis"})
		link, label := parts(w)
		// when
		test.Tap(link)
		// then
		assert.True(t, link.Visible())
		assert.False(t, label.Visible())
		xassert.Equal(t, "Deteis", link.Text)
		xassert.Equal(t, []int64{7}, iv.ShownBloodlines)
	})
	t.Run("shows link for location", func(t *testing.T) {
		// given
		iv := &testdouble.InfoViewerFake{}
		w := ui.NewInfoLink(iv)
		w.SetLocation(&app.EveLocation{ID: 60003760, Name: "Jita IV - Moon 4"})
		link, label := parts(w)
		// when
		test.Tap(link)
		// then
		assert.True(t, link.Visible())
		assert.False(t, label.Visible())
		xassert.Equal(t, "Jita IV - Moon 4", link.Text)
		xassert.Equal(t, []int64{60003760}, iv.ShownLocations)
	})
	t.Run("shows link for race", func(t *testing.T) {
		// given
		iv := &testdouble.InfoViewerFake{}
		w := ui.NewInfoLink(iv)
		w.SetRace(&app.EveRace{ID: 1, Name: "Caldari"})
		link, label := parts(w)
		// when
		test.Tap(link)
		// then
		assert.True(t, link.Visible())
		assert.False(t, label.Visible())
		xassert.Equal(t, "Caldari", link.Text)
		xassert.Equal(t, []int64{1}, iv.ShownRaces)
	})
	t.Run("shows placeholder after clear", func(t *testing.T) {
		// given
		w := ui.NewInfoLink(&testdouble.InfoViewerFake{})
		w.Set(&app.EveEntity{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter})
		// when
		w.Clear()
		// then
		link, label := parts(w)
		assert.False(t, link.Visible())
		assert.True(t, label.Visible())
	})
}
