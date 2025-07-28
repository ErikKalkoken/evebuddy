package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestEveCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		arg := storage.UpdateOrCreateEveCorporationParams{
			Description: "description",
			ID:          42,
			MemberCount: 888,
			Name:        "name",
			TaxRate:     0.12,
			Ticker:      "ticker",
			URL:         "url",
			WarEligible: false,
		}
		// when
		err := st.UpdateOrCreateEveCorporation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			got, err := st.GetEveCorporation(ctx, arg.ID)
			if assert.NoError(t, err) {
				assert.Nil(t, got.Alliance)
				assert.Nil(t, got.Faction)
				assert.Nil(t, got.Ceo)
				assert.Nil(t, got.Creator)
				assert.Nil(t, got.HomeStation)
				assert.EqualValues(t, arg.Description, got.Description)
				assert.EqualValues(t, arg.ID, got.ID)
				assert.EqualValues(t, arg.MemberCount, got.MemberCount)
				assert.EqualValues(t, arg.Name, got.Name)
				assert.EqualValues(t, arg.TaxRate, got.TaxRate)
				assert.EqualValues(t, arg.Ticker, got.Ticker)
				assert.EqualValues(t, arg.URL, got.URL)
				assert.EqualValues(t, arg.WarEligible, got.WarEligible)
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		alliance := factory.CreateEveEntityAlliance()
		faction := factory.CreateEveEntity()
		station := factory.CreateEveEntity()
		ceo := factory.CreateEveEntityCharacter()
		creator := factory.CreateEveEntityCharacter()
		dateFounded := factory.RandomTime()
		arg := storage.UpdateOrCreateEveCorporationParams{
			AllianceID:    optional.New(alliance.ID),
			CeoID:         optional.New(ceo.ID),
			CreatorID:     optional.New(creator.ID),
			DateFounded:   optional.New(dateFounded),
			Description:   "description",
			FactionID:     optional.New(faction.ID),
			HomeStationID: optional.New(station.ID),
			ID:            42,
			MemberCount:   888,
			Name:          "name",
			Shares:        optional.New(int64(987)),
			TaxRate:       0.12,
			Ticker:        "ticker",
			URL:           "url",
			WarEligible:   false,
		}
		// when
		err := st.UpdateOrCreateEveCorporation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			got, err := st.GetEveCorporation(ctx, arg.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, alliance, got.Alliance)
				assert.Equal(t, ceo, got.Ceo)
				assert.Equal(t, creator, got.Creator)
				assert.EqualValues(t, dateFounded, got.DateFounded.ValueOrZero())
				assert.EqualValues(t, arg.Description, got.Description)
				assert.Equal(t, faction, got.Faction)
				assert.Equal(t, station, got.HomeStation)
				assert.EqualValues(t, arg.ID, got.ID)
				assert.EqualValues(t, arg.MemberCount, got.MemberCount)
				assert.EqualValues(t, arg.Name, got.Name)
				assert.EqualValues(t, arg.TaxRate, got.TaxRate)
				assert.EqualValues(t, arg.Ticker, got.Ticker)
				assert.EqualValues(t, arg.Shares.ValueOrZero(), got.Shares.ValueOrZero())
				assert.EqualValues(t, arg.URL, got.URL)
				assert.EqualValues(t, arg.WarEligible, got.WarEligible)
			}
		}
	})
	t.Run("can update existing full", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
			ID: 42,
		})
		alliance := factory.CreateEveEntityAlliance()
		faction := factory.CreateEveEntity()
		station := factory.CreateEveEntity()
		ceo := factory.CreateEveEntityCharacter()
		creator := factory.CreateEveEntityCharacter()
		arg := storage.UpdateOrCreateEveCorporationParams{
			AllianceID:    optional.New(alliance.ID),
			CeoID:         optional.New(ceo.ID),
			CreatorID:     optional.New(creator.ID),
			Description:   "description",
			FactionID:     optional.New(faction.ID),
			HomeStationID: optional.New(station.ID),
			ID:            42,
			MemberCount:   888,
			Name:          "name",
			Shares:        optional.New(int64(987)),
			TaxRate:       0.12,
			Ticker:        "ticker",
			URL:           "url",
			WarEligible:   false,
		}
		// when
		err := st.UpdateOrCreateEveCorporation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			got, err := st.GetEveCorporation(ctx, arg.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, alliance, got.Alliance)
				assert.Equal(t, ceo, got.Ceo)
				assert.EqualValues(t, arg.Description, got.Description)
				assert.Equal(t, faction, got.Faction)
				assert.Equal(t, station, got.HomeStation)
				assert.EqualValues(t, arg.ID, got.ID)
				assert.EqualValues(t, arg.MemberCount, got.MemberCount)
				assert.EqualValues(t, arg.Name, got.Name)
				assert.EqualValues(t, arg.TaxRate, got.TaxRate)
				assert.EqualValues(t, arg.Ticker, got.Ticker)
				assert.EqualValues(t, arg.Shares.ValueOrZero(), got.Shares.ValueOrZero())
				assert.EqualValues(t, arg.URL, got.URL)
				assert.EqualValues(t, arg.WarEligible, got.WarEligible)
			}
		}
	})
	t.Run("can fetch by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateEveCorporation()
		// when
		c2, err := st.GetEveCorporation(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c1.Name, c2.Name)
		}
	})
	t.Run("can fetch character by ID with all fields populated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveCharacter()
		alliance := factory.CreateEveEntityAlliance()
		ceo := factory.CreateEveEntityCharacter()
		creator := factory.CreateEveEntityCharacter()
		faction := factory.CreateEveEntityWithCategory(app.EveEntityFaction)
		station := factory.CreateEveEntityWithCategory(app.EveEntityStation)
		founded := time.Now()
		c1 := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
			AllianceID:    optional.New(alliance.ID),
			CeoID:         optional.New(ceo.ID),
			CreatorID:     optional.New(creator.ID),
			DateFounded:   optional.New(founded),
			Description:   "description",
			FactionID:     optional.New(faction.ID),
			HomeStationID: optional.New(station.ID),
			MemberCount:   42,
			Name:          "name",
			Shares:        optional.New[int64](888),
			TaxRate:       0.1,
			Ticker:        "ticker",
			URL:           "url",
			WarEligible:   true,
		})
		// when
		c2, err := st.GetEveCorporation(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, alliance, c2.Alliance)
			assert.Equal(t, ceo, c2.Ceo)
			assert.Equal(t, creator, c2.Creator)
			assert.True(t, c2.DateFounded.MustValue().Equal(founded))
			assert.Equal(t, "description", c2.Description)
			assert.Equal(t, faction, c2.Faction)
			assert.Equal(t, station, c2.HomeStation)
			assert.EqualValues(t, 42, c2.MemberCount)
			assert.Equal(t, "name", c2.Name)
			assert.EqualValues(t, 888, c2.Shares.MustValue())
			assert.InDelta(t, 0.1, c2.TaxRate, 0.01)
			assert.Equal(t, "ticker", c2.Ticker)
			assert.Equal(t, "url", c2.URL)
			assert.Equal(t, c1.Name, c2.Name)
			assert.True(t, c2.WarEligible)
		}
	})
	t.Run("list corporation IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateEveCorporation()
		c2 := factory.CreateEveCorporation()
		// when
		got, err := st.ListEveCorporationIDs(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of(c1.ID, c2.ID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can update name", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateEveCorporation()
		// when
		err := st.UpdateEveCorporationName(ctx, c1.ID, "Alpha")
		// then
		if assert.NoError(t, err) {
			// assert.False(t, created)
			r, err := st.GetEveCorporation(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, "Alpha", r.Name)
			}
		}
	})
}
