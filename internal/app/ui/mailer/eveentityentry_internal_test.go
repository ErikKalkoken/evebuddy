package mailer

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
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

	t.Run("menu has disabled name item and actions", func(t *testing.T) {
		ee := &app.EveEntity{ID: 1, Name: "Bruce Wayne", Category: app.EveEntityCharacter}
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		w.showInfo = (&testdouble.InfoViewerFake{}).Show
		w.Set([]*app.EveEntity{ee})
		menu := w.makeBadgeMenu(ee)
		assert.Equal(t, []string{"Bruce Wayne (Character)", "", "Show Info", "Remove"}, menuLabels(menu))
		assert.True(t, menu.Items[0].Disabled)
		assert.Nil(t, menu.Items[0].Action)
		assert.Equal(t, theme.NewThemedResource(icons.AccountCircleSvg).Name(), menu.Items[0].Icon.Name())
	})
	t.Run("show info item shows info", func(t *testing.T) {
		// given
		iv := &testdouble.InfoViewerFake{}
		ee := &app.EveEntity{ID: 1, Name: "Wayne Enterprises", Category: app.EveEntityCorporation}
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		w.showInfo = iv.Show
		w.Set([]*app.EveEntity{ee})
		menu := w.makeBadgeMenu(ee)
		require.Equal(t, "Show Info", menu.Items[2].Label)
		assert.Equal(t, theme.NewThemedResource(icons.StarCircleOutlineSvg).Name(), menu.Items[0].Icon.Name())
		// when
		menu.Items[2].Action()
		// then
		assert.Equal(t, []*app.EveEntity{ee}, iv.Shown)
	})
	t.Run("mailing list has no show info item", func(t *testing.T) {
		ee := &app.EveEntity{ID: 1, Name: "Justice League", Category: app.EveEntityMailList}
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		w.showInfo = (&testdouble.InfoViewerFake{}).Show
		w.Set([]*app.EveEntity{ee})
		menu := w.makeBadgeMenu(ee)
		assert.Equal(t, []string{"Justice League (Mailing List)", "", "Remove"}, menuLabels(menu))
		assert.Equal(t, theme.MailComposeIcon().Name(), menu.Items[0].Icon.Name())
	})
	t.Run("remove item removes the entity", func(t *testing.T) {
		// given
		e1 := &app.EveEntity{ID: 1, Name: "Bruce Wayne", Category: app.EveEntityCharacter}
		e2 := &app.EveEntity{ID: 2, Name: "Clark Kent", Category: app.EveEntityCharacter}
		w := newEveEntityEntry(widget.NewLabel("To"), labelWith, loadIcon)
		w.Set([]*app.EveEntity{e1, e2})
		menu := w.makeBadgeMenu(e1)
		require.Equal(t, "Remove", menu.Items[2].Label)
		// when
		menu.Items[2].Action()
		// then
		assert.Equal(t, []*app.EveEntity{e2}, w.Items())
	})
	t.Run("read-only entry has no remove item", func(t *testing.T) {
		ee := &app.EveEntity{ID: 1, Name: "Bruce Wayne", Category: app.EveEntityCharacter}
		w := newEveEntityEntry(widget.NewLabel("From"), labelWith, loadIcon)
		w.readOnly = true
		w.showInfo = (&testdouble.InfoViewerFake{}).Show
		w.Set([]*app.EveEntity{ee})
		menu := w.makeBadgeMenu(ee)
		assert.Equal(t, []string{"Bruce Wayne (Character)", "", "Show Info"}, menuLabels(menu))
	})
	t.Run("read-only entry without actions has only name item", func(t *testing.T) {
		ee := &app.EveEntity{ID: 1, Name: "Justice League", Category: app.EveEntityMailList}
		w := newEveEntityEntry(widget.NewLabel("From"), labelWith, loadIcon)
		w.readOnly = true
		w.Set([]*app.EveEntity{ee})
		menu := w.makeBadgeMenu(ee)
		assert.Equal(t, []string{"Justice League (Mailing List)"}, menuLabels(menu))
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

func menuLabels(menu *fyne.Menu) []string {
	var s []string
	for _, it := range menu.Items {
		s = append(s, it.Label)
	}
	return s
}
