package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestMembers_ErrorDisplay(t *testing.T) {
	newMembers := func(t *testing.T, st *storage.Storage, factory testutil.Factory) *Members {
		a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		ec := factory.CreateEveCharacter()
		a.corporation.Store(factory.CreateCorporation(ec.Corporation.ID))
		return a
	}
	t.Run("shows error when update fails and keeps it on filtering", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := newMembers(t, st, factory)
		if _, err := db.Exec("DROP TABLE corporation_section_status"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")

		a.searchEntry.SetText("alpha")
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.footerLabel.Text, "Showing")
	})
	t.Run("hides error after successful update", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := newMembers(t, st, factory)
		a.errorLabel.SetText("ERROR: failed")
		a.errorLabel.Show()

		a.update(t.Context())
		assert.False(t, a.errorLabel.Visible())
	})
}

func TestMembers_ResetClearsFooter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	}))
	a.rows = []memberRow{{id: 1, name: "Alpha"}, {id: 2, name: "Bravo"}}
	var pending []func() // run background work later, like the real app
	orig := runAsync
	runAsync = func(f func()) { pending = append(pending, f) }
	t.Cleanup(func() { runAsync = orig })
	a.filterRowsAsync() // a filter still running from before

	a.update(t.Context()) // no corporation
	for _, f := range pending {
		f()
	}

	assert.Equal(t, "", a.footerLabel.Text)
	assert.Empty(t, a.rowsFiltered)
}
