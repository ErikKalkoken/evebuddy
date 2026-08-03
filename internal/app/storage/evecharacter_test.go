package storage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveCharacter(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			ID   = 42
			name = "Erik"
		)
		corporation := f.CreateEveEntityCorporation()
		bloodline := f.CreateEveBloodline()
		race := bloodline.Race
		// when
		err := st.UpdateOrCreateEveCharacter(t.Context(), storage.CreateEveCharacterParams{
			ID:            ID,
			Name:          name,
			CorporationID: corporation.ID,
			RaceID:        race.ID,
			BloodlineID:   bloodline.ID,
		})
		// then
		require.NoError(t, err)
		o, err := st.GetEveCharacter(t.Context(), ID)
		require.NoError(t, err)
		xassert.Equal(t, name, o.Name)
		xassert.Equal(t, ID, o.ID)
		xassert.Equal(t, corporation, o.Corporation)
		xassert.Equal(t, bloodline.ID, o.Bloodline.MustValue().ID)
		xassert.Equal(t, race, o.Race)
	})

	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		const (
			ID               = 42
			name             = "Erik"
			gender           = "male"
			securityStatus   = -5.2
			corporationTitle = "corporationTitle"
			description      = "description"
		)
		corporation := f.CreateEveEntityCorporation()
		bloodline := f.CreateEveBloodline()
		race := bloodline.Race
		birthday := time.Now().Add(-100 * time.Hour).Round(time.Second)
		alliance := f.CreateEveEntityAlliance()
		faction := f.CreateEveEntityFaction()

		// when
		err := st.UpdateOrCreateEveCharacter(t.Context(), storage.CreateEveCharacterParams{
			AllianceID:       optional.New(alliance.ID),
			Birthday:         birthday,
			BloodlineID:      bloodline.ID,
			CorporationID:    corporation.ID,
			CorporationTitle: optional.New(corporationTitle),
			Description:      optional.New(description),
			FactionID:        optional.New(faction.ID),
			Gender:           gender,
			ID:               ID,
			Name:             name,
			RaceID:           race.ID,
			SecurityStatus:   optional.New(securityStatus),
		})

		// then
		require.NoError(t, err)
		o, err := st.GetEveCharacter(t.Context(), ID)
		require.NoError(t, err)
		xassert.Equal(t, alliance, o.Alliance.MustValue())
		xassert.Equal(t, birthday, o.Birthday)
		xassert.Equal(t, bloodline.ID, o.Bloodline.MustValue().ID)
		xassert.Equal(t, corporation, o.Corporation)
		xassert.Equal(t, corporationTitle, o.CorporationTitle.MustValue())
		xassert.Equal(t, description, o.Description.MustValue())
		xassert.Equal(t, faction, o.Faction.MustValue())
		xassert.Equal(t, gender, o.Gender)
		xassert.Equal(t, ID, o.ID)
		xassert.Equal(t, name, o.Name)
		xassert.Equal(t, securityStatus, o.SecurityStatus.MustValue())
	})

	t.Run("can update existing 1", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := f.CreateEveCharacter()
		const (
			name             = "Erik"
			gender           = "male"
			securityStatus   = -5.2
			corporationTitle = "corporationTitle"
			description      = "description"
		)
		corporation := f.CreateEveEntityCorporation()
		bloodline := f.CreateEveBloodline()
		race := bloodline.Race
		birthday := time.Now().Add(-100 * time.Hour).Round(time.Second)
		alliance := f.CreateEveEntityAlliance()
		faction := f.CreateEveEntityFaction()

		// when
		err := st.UpdateOrCreateEveCharacter(t.Context(), storage.CreateEveCharacterParams{
			AllianceID:       optional.New(alliance.ID),
			Birthday:         birthday,
			BloodlineID:      bloodline.ID,
			CorporationID:    corporation.ID,
			CorporationTitle: optional.New(corporationTitle),
			Description:      optional.New(description),
			FactionID:        optional.New(faction.ID),
			Gender:           gender,
			ID:               c1.ID,
			Name:             name,
			RaceID:           race.ID,
			SecurityStatus:   optional.New(float64(securityStatus)),
		})

		// then
		require.NoError(t, err)
		c2, err := st.GetEveCharacter(t.Context(), c1.ID)
		require.NoError(t, err)
		xassert.Equal(t, alliance, c2.Alliance.MustValue())
		xassert.Equal(t, birthday, c2.Birthday)
		xassert.Equal(t, bloodline.ID, c2.Bloodline.MustValue().ID)
		xassert.Equal(t, corporation, c2.Corporation)
		xassert.Equal(t, corporationTitle, c2.CorporationTitle.MustValue())
		xassert.Equal(t, description, c2.Description.MustValue())
		xassert.Equal(t, faction, c2.Faction.MustValue())
		xassert.Equal(t, gender, c2.Gender)
		xassert.Equal(t, name, c2.Name)
		xassert.Equal(t, securityStatus, c2.SecurityStatus.MustValue())
	})

	t.Run("can update existing with neutral security status", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := f.CreateEveCharacter()
		const (
			securityStatus = 0
		)

		// when
		err := st.UpdateOrCreateEveCharacter(t.Context(), storage.CreateEveCharacterParams{
			Birthday:         c1.Birthday,
			BloodlineID:      c1.Bloodline.MustValue().ID,
			CorporationID:    c1.Corporation.ID,
			CorporationTitle: c1.CorporationTitle,
			Description:      c1.Description,
			Gender:           c1.Gender,
			ID:               c1.ID,
			Name:             c1.Name,
			RaceID:           c1.Race.ID,
			SecurityStatus:   optional.New[float64](securityStatus),
		})

		// then
		require.NoError(t, err)
		c2, err := st.GetEveCharacter(t.Context(), c1.ID)
		require.NoError(t, err)
		xassert.EqualOptional(t, securityStatus, c2.SecurityStatus)
	})

	t.Run("can delete", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := f.CreateEveCharacter()
		// when
		err := st.DeleteEveCharacter(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		_, err2 := st.GetEveCharacter(t.Context(), c.ID)
		assert.ErrorIs(t, err2, app.ErrNotFound)
	})

	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetEveCharacter(t.Context(), 99)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})

	t.Run("can fetch character by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := f.CreateEveCharacter()
		// when
		c2, err := st.GetEveCharacter(t.Context(), c1.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, c1.Birthday.UTC(), c2.Birthday.UTC())
		xassert.Equal(t, c1.Corporation, c2.Corporation)
		xassert.Equal(t, c1.Description, c2.Description)
		xassert.Equal(t, c1.Gender, c2.Gender)
		xassert.Equal(t, c1.ID, c2.ID)
		xassert.Equal(t, c1.Name, c2.Name)
		xassert.Equal(t, c1.Race, c2.Race)
		xassert.Equal(t, c1.SecurityStatus, c2.SecurityStatus)
		xassert.Equal(t, c1.CorporationTitle, c2.CorporationTitle)
		xassert.Empty(t, c2.Alliance)
		xassert.Empty(t, c2.Faction)
	})

	t.Run("can fetch character by ID with all fields populated", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		f.CreateEveCharacter()
		alliance := f.CreateEveEntityAlliance()
		faction := f.CreateEveEntity(app.EveEntity{Category: app.EveEntityFaction})
		arg := storage.CreateEveCharacterParams{
			AllianceID: optional.New(alliance.ID),
			FactionID:  optional.New(faction.ID),
		}
		c1 := f.CreateEveCharacter(arg)
		// when
		c2, err := st.GetEveCharacter(t.Context(), c1.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, alliance, c2.Alliance.ValueOrZero())
		xassert.Equal(t, c1.Birthday.UTC(), c2.Birthday.UTC())
		xassert.Equal(t, c1.Corporation, c2.Corporation)
		xassert.Equal(t, c1.Description, c2.Description)
		xassert.Equal(t, faction, c2.Faction.ValueOrZero())
		xassert.Equal(t, c1.ID, c2.ID)
		xassert.Equal(t, c1.Name, c2.Name)
	})

	t.Run("can update name", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := f.CreateEveCharacter()
		// when
		err := st.UpdateEveCharacterName(t.Context(), c1.ID, "Erik")
		// then
		require.NoError(t, err)
		c2, err := st.GetEveCharacter(t.Context(), c1.ID)
		require.NoError(t, err)
		xassert.Equal(t, "Erik", c2.Name)
	})
}
