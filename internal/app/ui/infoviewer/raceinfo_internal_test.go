package infoviewer

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestRaceInfo_update(t *testing.T) {
	t.Run("should show race with faction", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		f := factory.CreateEveEntityFaction()
		r := factory.CreateEveRace(storage.UpdateOrCreateEveRaceParams{
			Description: "My race",
			FactionID:   optional.New(f.ID),
		})
		a := newRaceInfo(New(u), r.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, r.Name, a.name.Text)
		assert.Equal(t, "My race", a.description.Text)
		assert.Subset(t, tabLabelTexts(t, a.tabs, "Attributes"), []string{"Faction", f.Name})
	})
}
