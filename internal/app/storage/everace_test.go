package storage_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveRace(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			raceID      = 42
			description = "description"
			name        = "name"
		)

		// when
		err := st.UpdateOrCreateEveRace(t.Context(), storage.UpdateOrCreateEveRaceParams{
			ID:          raceID,
			Description: description,
			Name:        name,
		})

		// then
		require.NoError(t, err)
		x, err := st.GetEveRace(t.Context(), raceID)
		require.NoError(t, err)
		xassert.Equal(t, raceID, x.ID)
		xassert.Equal(t, description, x.Description)
		xassert.Equal(t, name, x.Name)
	})

	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			raceID      = 42
			description = "description"
			name        = "name"
		)
		faction := f.CreateEveEntityFaction()

		// when
		err := st.UpdateOrCreateEveRace(t.Context(), storage.UpdateOrCreateEveRaceParams{
			ID:          raceID,
			Description: description,
			Name:        name,
			FactionID:   optional.New(faction.ID),
		})

		// then
		require.NoError(t, err)
		x, err := st.GetEveRace(t.Context(), raceID)
		require.NoError(t, err)
		xassert.Equal(t, raceID, x.ID)
		xassert.Equal(t, description, x.Description)
		xassert.Equal(t, name, x.Name)
		xassert.EqualOptional(t, faction, x.Faction)
	})

	t.Run("can update existing", func(t *testing.T) {
		x1 := f.CreateEveRace()
		const (
			description = "description"
			name        = "name"
		)
		faction := f.CreateEveEntityFaction()

		// when
		err := st.UpdateOrCreateEveRace(t.Context(), storage.UpdateOrCreateEveRaceParams{
			ID:          x1.ID,
			Description: description,
			Name:        name,
			FactionID:   optional.New(faction.ID),
		})

		// then
		require.NoError(t, err)
		x, err := st.GetEveRace(t.Context(), x1.ID)
		require.NoError(t, err)
		xassert.Equal(t, x1.ID, x.ID)
		xassert.Equal(t, description, x.Description)
		xassert.Equal(t, name, x.Name)
		xassert.EqualOptional(t, faction, x.Faction)
	})
}
