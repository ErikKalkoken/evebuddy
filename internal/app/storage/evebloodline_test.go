package storage_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveBloodline(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			id          = 42
			description = "description"
			name        = "name"
		)
		corporation := f.CreateEveEntityCorporation()
		race := f.CreateEveRace()
		// when
		arg := storage.CreateEveBloodlineParams{
			ID:            id,
			Description:   description,
			Name:          name,
			CorporationID: corporation.ID,
			RaceID:        race.ID,
		}
		err := st.CreateEveBloodline(t.Context(), arg)
		// then
		require.NoError(t, err)
		eb, err := st.GetEveBloodline(t.Context(), id)
		require.NoError(t, err)
		xassert.Equal(t, corporation, eb.Corporation)
		xassert.Equal(t, description, eb.Description)
		xassert.Equal(t, id, eb.ID)
		xassert.Equal(t, name, eb.Name)
		xassert.Equal(t, race, eb.Race)
		xassert.Empty(t, eb.Charisma)
		xassert.Empty(t, eb.Intelligence)
		xassert.Empty(t, eb.Memory)
		xassert.Empty(t, eb.Perception)
		xassert.Empty(t, eb.Willpower)
		xassert.Empty(t, eb.ShipTypeID)
	})

	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			charisma     = 11
			description  = "description"
			id           = 42
			intelligence = 12
			memory       = 13
			name         = "name"
			perception   = 14
			willpower    = 15
		)
		corporation := f.CreateEveEntityCorporation()
		race := f.CreateEveRace()
		shipType := f.CreateEveType()
		// when
		arg := storage.CreateEveBloodlineParams{
			Charisma:      optional.New[int64](charisma),
			CorporationID: corporation.ID,
			Description:   description,
			ID:            id,
			Intelligence:  optional.New[int64](intelligence),
			Memory:        optional.New[int64](memory),
			Name:          name,
			Perception:    optional.New[int64](perception),
			RaceID:        race.ID,
			ShipTypeID:    optional.New(shipType.ID),
			Willpower:     optional.New[int64](willpower),
		}
		err := st.CreateEveBloodline(t.Context(), arg)
		// then
		require.NoError(t, err)
		eb, err := st.GetEveBloodline(t.Context(), id)
		require.NoError(t, err)
		xassert.Equal(t, corporation, eb.Corporation)
		xassert.Equal(t, description, eb.Description)
		xassert.Equal(t, id, eb.ID)
		xassert.Equal(t, name, eb.Name)
		xassert.Equal(t, race, eb.Race)
		xassert.EqualOptional(t, charisma, eb.Charisma)
		xassert.EqualOptional(t, intelligence, eb.Intelligence)
		xassert.EqualOptional(t, memory, eb.Memory)
		xassert.EqualOptional(t, perception, eb.Perception)
		xassert.EqualOptional(t, willpower, eb.Willpower)
	})
}
