package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestColonyInputList(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	// row returns the rendered row at id.
	row := func(a *colonyInputList, id int) *colonyTypeItemWidget {
		co := a.list.CreateItem()
		a.list.UpdateItem(id, co)
		return co.(*colonyTypeItemWidget)
	}

	t.Run("should sort inputs by name", func(t *testing.T) {
		a := newColonyInputList(u)
		a.set([]colonyInputItem{
			{typeID: 2, name: "Bravo", demand: 40, isRouted: true},
			{typeID: 1, name: "Alpha", demand: 40, isRouted: true},
		}, "No inputs")
		assert.Equal(t, "Alpha", a.items[0].name)
		assert.Equal(t, "Bravo", a.items[1].name)
		assert.False(t, a.empty.Visible())
	})
	t.Run("should show routed input normally", func(t *testing.T) {
		a := newColonyInputList(u)
		a.set([]colonyInputItem{{typeID: 1, name: "Alpha", demand: 3_000, inStock: 1_200, isRouted: true}}, "No inputs")
		w := row(a, 0)
		assert.Equal(t, "Alpha", w.name.Text)
		assert.Equal(t, "Needs 3,000 • 1,200 in stock", w.details.Text)
		assert.Equal(t, widget.MediumImportance, w.details.Importance)
	})
	t.Run("should show unrouted input as danger", func(t *testing.T) {
		a := newColonyInputList(u)
		a.set([]colonyInputItem{{typeID: 1, name: "Alpha", demand: 3_000}}, "No inputs")
		w := row(a, 0)
		assert.Equal(t, "Needs 3,000 • 0 in stock • not routed", w.details.Text)
		assert.Equal(t, widget.DangerImportance, w.details.Importance)
	})
	t.Run("should show empty text when there are no inputs", func(t *testing.T) {
		a := newColonyInputList(u)
		a.set([]colonyInputItem{}, "No schematic")
		assert.True(t, a.empty.Visible())
		assert.Equal(t, "No schematic", a.empty.Text)
	})
}
