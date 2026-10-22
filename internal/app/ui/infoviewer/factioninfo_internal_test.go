package infoviewer

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestFactionInfo_update(t *testing.T) {
	t.Run("should show faction", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		f := factory.CreateEveFaction(storage.CreateEveFactionParams{
			Description:        "My faction",
			IsUnique:           true,
			StationCount:       1234,
			StationSystemCount: 56,
		})
		a := newFactionInfo(New(u), f.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, f.Name, a.name.Text)
		assert.Equal(t, "My faction", a.description.Text)
		assert.Subset(t, tabLabelTexts(t, a.tabs, "Attributes"), []string{
			"Unique", "yes",
			"Stations", "1234",
			"Systems with station", "56",
			"Corporation", f.Corporation.MustValue().Name,
			"Solar System", f.SolarSystem.MustValue().Name,
		})
	})
}
