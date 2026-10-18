package screens

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestCorporationWealth_ErrorDisplay(t *testing.T) {
	t.Run("shows error and clears charts when update fails", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.corporation.Store(factory.CreateCorporation())
		a.categorySplitTitleLabel.SetText("Total Net Worth By Category - Total: 1.0 B")
		a.walletsTitleLabel.SetText("Wallet Balance by Division")
		a.assetsTitleLabel.SetText("Asset Value by Division")
		if _, err := db.Exec("DROP TABLE corporation_section_status"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.statusLabel.Visible())
		assert.Contains(t, a.statusLabel.Text, "Failed to fetch data")
		assert.Empty(t, a.categorySplitTitleLabel.Text)
		assert.Empty(t, a.walletsTitleLabel.Text)
		assert.Empty(t, a.assetsTitleLabel.Text)
	})
	t.Run("shows error on desktop", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		w := test.NewWindow(a)
		defer w.Close()
		a.corporation.Store(factory.CreateCorporation())
		if _, err := db.Exec("DROP TABLE corporation_section_status"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.statusLabel.Visible())
		assert.Contains(t, a.statusLabel.Text, "Failed to fetch data")
		assert.True(t, slices.Contains(test.LaidOutObjects(a), fyne.CanvasObject(a.statusLabel)))
	})
	t.Run("shows error on mobile", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewCorporationWealth(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: true,
			Storage:  st,
		}))
		w := test.NewWindow(a)
		defer w.Close()
		a.corporation.Store(factory.CreateCorporation())
		if _, err := db.Exec("DROP TABLE corporation_section_status"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.statusLabel.Visible())
		assert.Contains(t, a.statusLabel.Text, "Failed to fetch data")
		assert.True(t, slices.Contains(test.LaidOutObjects(a), fyne.CanvasObject(a.statusLabel)))
	})
}
