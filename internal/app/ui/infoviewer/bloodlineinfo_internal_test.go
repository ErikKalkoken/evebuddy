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

func TestBloodlineInfo_update(t *testing.T) {
	t.Run("should show bloodline", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		ship := factory.CreateEveType()
		b := factory.CreateEveBloodline(storage.CreateEveBloodlineParams{
			Description: "My bloodline",
			ShipTypeID:  optional.New(ship.ID),
		})
		a := newBloodlineInfo(New(u), b.ID)
		// when
		err := a.update(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, b.Name, a.name.Text)
		assert.Equal(t, "My bloodline", a.description.Text)
		assert.Subset(t, tabLabelTexts(t, a.tabs, "Attributes"), []string{
			"Race", b.Race.Name,
			"Starter corporation", b.Corporation.Name,
			"Starter ship", ship.Name,
		})
	})
}
