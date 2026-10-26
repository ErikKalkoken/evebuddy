package mailer

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestEveEntityEntry(t *testing.T) {
	test.NewTempApp(t)
	loadIcon := testutil.NewEveImageServiceStub().EveEntityLogoAsync
	e1 := &app.EveEntity{ID: 1, Name: "Bruce Wayne", Category: app.EveEntityCharacter}
	e2 := &app.EveEntity{ID: 2, Name: "Clark Kent", Category: app.EveEntityCharacter}

	t.Run("can add item", func(t *testing.T) {
		// given
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		w.Set([]*app.EveEntity{e1})
		// when
		w.Add(e2)
		// then
		assert.Equal(t, []*app.EveEntity{e1, e2}, w.Items())
	})
	t.Run("does not add existing item again", func(t *testing.T) {
		// given
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		w.Set([]*app.EveEntity{e1})
		// when
		w.Add(&app.EveEntity{ID: 1, Name: "Bruce Wayne", Category: app.EveEntityCharacter})
		// then
		assert.Equal(t, []*app.EveEntity{e1}, w.Items())
	})
	t.Run("can remove item", func(t *testing.T) {
		// given
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		w.Set([]*app.EveEntity{e1, e2})
		// when
		w.Remove(e1.ID)
		// then
		assert.Equal(t, []*app.EveEntity{e2}, w.Items())
	})
	t.Run("removing unknown item does nothing", func(t *testing.T) {
		// given
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		w.Set([]*app.EveEntity{e1})
		// when
		w.Remove(99)
		// then
		assert.Equal(t, []*app.EveEntity{e1}, w.Items())
	})
	t.Run("string lists all names", func(t *testing.T) {
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		w.Set([]*app.EveEntity{e1, e2})
		assert.Equal(t, "Bruce Wayne, Clark Kent", w.String())
		assert.False(t, w.IsEmpty())
	})
	t.Run("string is empty without items", func(t *testing.T) {
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		assert.Equal(t, "", w.String())
		assert.True(t, w.IsEmpty())
	})
}

func TestEveEntityEntry_BadgeMenu(t *testing.T) {
	test.NewTempApp(t)
	loadIcon := testutil.NewEveImageServiceStub().EveEntityLogoAsync

	t.Run("name item of character shows info", func(t *testing.T) {
		// given
		iv := &testdouble.InfoViewerFake{}
		ee := &app.EveEntity{ID: 1, Name: "Bruce Wayne", Category: app.EveEntityCharacter}
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		w.showInfo = iv.Show
		w.Set([]*app.EveEntity{ee})
		menu := w.makeBadgeMenu(ee)
		require.NotNil(t, menu.Items[0].Action)
		// when
		menu.Items[0].Action()
		// then
		assert.Equal(t, "Bruce Wayne (Character)", menu.Items[0].Label)
		assert.Equal(t, []*app.EveEntity{ee}, iv.Shown)
	})
	t.Run("name item of other categories has no action", func(t *testing.T) {
		iv := &testdouble.InfoViewerFake{}
		ee := &app.EveEntity{ID: 1, Name: "Wayne Enterprises", Category: app.EveEntityCorporation}
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		w.showInfo = iv.Show
		w.Set([]*app.EveEntity{ee})
		menu := w.makeBadgeMenu(ee)
		assert.Nil(t, menu.Items[0].Action)
	})
	t.Run("remove item removes the entity", func(t *testing.T) {
		// given
		e1 := &app.EveEntity{ID: 1, Name: "Bruce Wayne", Category: app.EveEntityCharacter}
		e2 := &app.EveEntity{ID: 2, Name: "Clark Kent", Category: app.EveEntityCharacter}
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		w.Set([]*app.EveEntity{e1, e2})
		menu := w.makeBadgeMenu(e1)
		require.False(t, menu.Items[2].Disabled)
		// when
		menu.Items[2].Action()
		// then
		assert.Equal(t, []*app.EveEntity{e2}, w.Items())
	})
	t.Run("remove item is disabled when entry is disabled", func(t *testing.T) {
		ee := &app.EveEntity{ID: 1, Name: "Bruce Wayne", Category: app.EveEntityCharacter}
		w := newEveEntityEntry(widget.NewLabel("From"), labelWith, loadIcon)
		w.Set([]*app.EveEntity{ee})
		w.Disable()
		menu := w.makeBadgeMenu(ee)
		assert.True(t, menu.Items[2].Disabled)
	})
	t.Run("tapping a badge shows its menu", func(t *testing.T) {
		// given
		ee := &app.EveEntity{ID: 1, Name: "Bruce Wayne", Category: app.EveEntityCharacter}
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		w.Set([]*app.EveEntity{ee})
		win := test.NewWindow(w)
		defer win.Close()
		var badge *eveEntityBadge
		for _, o := range test.LaidOutObjects(w) {
			if x, ok := o.(*eveEntityBadge); ok {
				badge = x
			}
		}
		require.NotNil(t, badge)
		// when
		test.Tap(badge)
		// then
		assert.NotNil(t, win.Canvas().Overlays().Top())
	})
}
