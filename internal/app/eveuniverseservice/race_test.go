package eveuniverseservice_test

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestGetOrCreateEveRaceESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})

	t.Run("should return existing race", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		x1 := f.CreateEveRace()
		// when
		x2, err := s.GetOrCreateRaceESI(t.Context(), x1.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})

	t.Run("should create race from ESI when it does not exit in DB", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		const (
			name        = "name"
			description = "description"
			raceID      = 7
		)
		faction := f.CreateEveEntityFaction()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/universe/races",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id": faction.ID,
					"description": description,
					"name":        name,
					"race_id":     raceID,
				},
			}))

		// when

		x1, err := s.GetOrCreateRaceESI(t.Context(), raceID)

		// then
		require.NoError(t, err)
		xassert.Equal(t, name, x1.Name)
		xassert.Equal(t, description, x1.Description)
		if xassert.NotEmpty(t, x1.Faction) {
			xassert.Equal(t, faction, x1.Faction.MustValue())
		}
		x2, err := st.GetEveRace(t.Context(), raceID)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})

	t.Run("should update race from ESI when it has no faction", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		const (
			name        = "name"
			description = "description"
			raceID      = 7
		)
		err := st.UpdateOrCreateEveRace(t.Context(), storage.UpdateOrCreateEveRaceParams{
			Description: description,
			ID:          raceID,
			Name:        name,
		})
		require.NoError(t, err)
		faction := f.CreateEveEntityFaction()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/universe/races",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id": faction.ID,
					"description": description,
					"name":        name,
					"race_id":     raceID,
				},
			}))

		// when

		x2, err := s.GetOrCreateRaceESI(t.Context(), raceID)

		// then
		require.NoError(t, err)
		xassert.Equal(t, name, x2.Name)
		xassert.Equal(t, description, x2.Description)
		if xassert.NotEmpty(t, x2.Faction) {
			xassert.Equal(t, faction, x2.Faction.MustValue())
		}
		x3, err := st.GetEveRace(t.Context(), raceID)
		require.NoError(t, err)
		xassert.Equal(t, x2, x3)
	})

	t.Run("should return specific error when race ID is invalid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/universe/races",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id": 500001,
					"description": "Founded on the tenets of patriotism and hard work...",
					"name":        "Caldari",
					"race_id":     7,
				},
			}))

		// when
		_, err := s.GetOrCreateRaceESI(t.Context(), 42)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestGetOrCreateEveBloodlineESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{Storage: st})

	t.Run("should return existing bloodline", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		x1 := f.CreateEveBloodline()
		// when
		x2, err := s.GetOrCreateBloodlineESI(t.Context(), x1.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})

	t.Run("should create normal bloodline from ESI when it does not exit", func(t *testing.T) {
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
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/universe/bloodlines",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"bloodline_id":   id,
					"charisma":       charisma,
					"corporation_id": corporation.ID,
					"description":    description,
					"intelligence":   intelligence,
					"memory":         memory,
					"name":           name,
					"perception":     perception,
					"race_id":        race.ID,
					"ship_type_id":   shipType.ID,
					"willpower":      willpower,
				},
			}))

		// when
		x1, err := s.GetOrCreateBloodlineESI(t.Context(), id)

		// then
		require.NoError(t, err)
		xassert.Equal(t, corporation, x1.Corporation)
		xassert.Equal(t, description, x1.Description)
		xassert.Equal(t, id, x1.ID)
		xassert.Equal(t, name, x1.Name)
		xassert.Equal(t, race, x1.Race)
		xassert.Equal(t, charisma, x1.Charisma.MustValue())
		xassert.Equal(t, intelligence, x1.Intelligence.MustValue())
		xassert.Equal(t, memory, x1.Memory.MustValue())
		xassert.Equal(t, perception, x1.Perception.MustValue())
		xassert.Equal(t, willpower, x1.Willpower.MustValue())
		if xassert.NotEmpty(t, x1.ShipTypeID) {
			xassert.Equal(t, shipType.ID, x1.ShipTypeID.MustValue())
		}
		x2, err := st.GetEveBloodline(t.Context(), id)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})

	t.Run("should create minimal bloodline from ESI when it does not exit", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			description = "description"
			id          = 42
			name        = "name"
		)
		corporation := f.CreateEveEntityCorporation()
		race := f.CreateEveRace()
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/universe/bloodlines",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"bloodline_id":   id,
					"charisma":       0,
					"corporation_id": corporation.ID,
					"description":    description,
					"intelligence":   0,
					"memory":         0,
					"name":           name,
					"perception":     0,
					"race_id":        race.ID,
					"ship_type_id":   nil,
					"willpower":      0,
				},
			}))

		// when
		x1, err := s.GetOrCreateBloodlineESI(t.Context(), id)

		// then
		require.NoError(t, err)
		xassert.Equal(t, corporation, x1.Corporation)
		xassert.Equal(t, description, x1.Description)
		xassert.Equal(t, id, x1.ID)
		xassert.Equal(t, name, x1.Name)
		xassert.Equal(t, race, x1.Race)
		xassert.Empty(t, x1.Charisma)
		xassert.Empty(t, x1.Intelligence)
		xassert.Empty(t, x1.Memory)
		xassert.Empty(t, x1.Perception)
		xassert.Empty(t, x1.Willpower)
		xassert.Empty(t, x1.ShipTypeID)
		x2, err := st.GetEveBloodline(t.Context(), id)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})

	t.Run("should return specific error when bloodline ID is invalid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/universe/bloodlines",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"bloodline_id":   11,
					"charisma":       3,
					"corporation_id": 1000014,
					"description":    "Achura has been part of the Caldari State for three centuries, and yet their culture has always remained something of a mystery. Originally from the Saisio System, they are reclusive and introverted, and show little interest in the ephemeral phenomena of the material world. Intensely spiritual, Achura pilots have only recently taken to the stars, driven in large part by a desire to unlock the secrets of the universe.",
					"intelligence":   8,
					"memory":         6,
					"name":           "Achura",
					"perception":     7,
					"race_id":        1,
					"ship_type_id":   601,
					"willpower":      6,
				},
			}))

		// when
		_, err := s.GetOrCreateBloodlineESI(t.Context(), 42)

		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}
