package storage_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveFaction(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			id                 = 42
			description        = "description"
			name               = "name"
			isUnique           = true
			sizeFactor         = 4
			stationCount       = 99
			stationSystemCount = 7
		)
		arg := storage.CreateEveFactionParams{
			ID:                 id,
			Description:        description,
			Name:               name,
			IsUnique:           isUnique,
			SizeFactor:         sizeFactor,
			StationCount:       stationCount,
			StationSystemCount: stationSystemCount,
		}

		// when
		err := st.CreateEveFaction(t.Context(), arg)

		// then
		require.NoError(t, err)
		ef, err := st.GetEveFaction(t.Context(), id)
		require.NoError(t, err)
		xassert.Equal(t, description, ef.Description)
		xassert.Equal(t, id, ef.ID)
		xassert.Equal(t, name, ef.Name)
		xassert.Equal(t, isUnique, ef.IsUnique)
		xassert.Equal(t, sizeFactor, ef.SizeFactor)
		xassert.Equal(t, stationCount, ef.StationCount)
		xassert.Equal(t, stationSystemCount, ef.StationSystemCount)
		xassert.Empty(t, ef.Corporation)
		xassert.Empty(t, ef.MilitiaCorporation)
		xassert.Empty(t, ef.SolarSystem)
	})

	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			factionID          = 42
			description        = "description"
			name               = "name"
			isUnique           = true
			sizeFactor         = 4
			stationCount       = 99
			stationSystemCount = 7
		)
		corporation := f.CreateEveEntityCorporation()
		militaryCorporation := f.CreateEveEntityCorporation()
		solarSystem := f.CreateEveSolarSystem()
		arg := storage.CreateEveFactionParams{
			ID:                   factionID,
			Description:          description,
			Name:                 name,
			IsUnique:             isUnique,
			SizeFactor:           sizeFactor,
			StationCount:         stationCount,
			StationSystemCount:   stationSystemCount,
			CorporationID:        optional.New(corporation.ID),
			MilitiaCorporationID: optional.New(militaryCorporation.ID),
			SolarSystemID:        optional.New(solarSystem.ID),
		}

		// when
		err := st.CreateEveFaction(t.Context(), arg)

		// then
		require.NoError(t, err)
		ef, err := st.GetEveFaction(t.Context(), factionID)
		require.NoError(t, err)
		xassert.Equal(t, description, ef.Description)
		xassert.Equal(t, factionID, ef.ID)
		xassert.Equal(t, name, ef.Name)
		xassert.Equal(t, isUnique, ef.IsUnique)
		xassert.Equal(t, sizeFactor, ef.SizeFactor)
		xassert.Equal(t, stationCount, ef.StationCount)
		xassert.Equal(t, stationSystemCount, ef.StationSystemCount)
		xassert.EqualOptional(t, corporation, ef.Corporation)
		xassert.EqualOptional(t, militaryCorporation, ef.MilitiaCorporation)
		xassert.EqualOptional(t, solarSystem.ToEveEntity(), ef.SolarSystem)
	})
}
