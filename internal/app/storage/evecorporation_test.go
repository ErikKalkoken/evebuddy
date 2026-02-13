package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		arg := storage.UpdateOrCreateEveCorporationParams{
			Description: optional.New("description"),
			ID:          42,
			MemberCount: 888,
			Name:        "name",
			TaxRate:     0.12,
			Ticker:      "ticker",
			URL:         optional.New("url"),
			WarEligible: optional.New(false),
		}
		// when
		err := st.UpdateOrCreateEveCorporation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			got, err := st.GetEveCorporation(ctx, arg.ID)
			if assert.NoError(t, err) {
				assert.True(t, got.Alliance.IsEmpty())
				assert.True(t, got.Faction.IsEmpty())
				assert.True(t, got.Ceo.IsEmpty())
				assert.True(t, got.Creator.IsEmpty())
				assert.Empty(t, got.HomeStation)
				xassert.Equal(t, arg.Description.ValueOrZero(), got.Description)
				xassert.Equal(t, arg.ID, got.ID)
				xassert.Equal(t, arg.MemberCount, got.MemberCount)
				xassert.Equal(t, arg.Name, got.Name)
				xassert.Equal(t, arg.TaxRate, got.TaxRate)
				xassert.Equal(t, arg.Ticker, got.Ticker)
				xassert.Equal(t, arg.URL.ValueOrZero(), got.URL.ValueOrZero())
				xassert.Equal(t, arg.WarEligible.ValueOrZero(), got.WarEligible.ValueOrZero())
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
			Description:   optional.New("description"),
			FactionID:     optional.New(faction.ID),
			HomeStationID: optional.New(station.ID),
			ID:            42,
			MemberCount:   888,
			Name:          "name",
			Shares:        optional.New(int64(987)),
			TaxRate:       0.12,
			Ticker:        "ticker",
			URL:           optional.New("url"),
			WarEligible:   optional.New(false),
		}
		// when
		err := st.UpdateOrCreateEveCorporation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			got, err := st.GetEveCorporation(ctx, arg.ID)
			if assert.NoError(t, err) {
				xassert.Equal(t, alliance, got.Alliance.ValueOrZero())
				xassert.Equal(t, ceo, got.Ceo.ValueOrZero())
				xassert.Equal(t, creator, got.Creator.ValueOrZero())
				xassert.Equal(t, dateFounded, got.DateFounded.ValueOrZero())
				xassert.Equal(t, arg.Description.ValueOrZero(), got.Description)
				xassert.Equal(t, faction, got.Faction.ValueOrZero())
				xassert.Equal(t, station, got.HomeStation.ValueOrZero())
				xassert.Equal(t, arg.ID, got.ID)
				xassert.Equal(t, arg.MemberCount, got.MemberCount)
				xassert.Equal(t, arg.Name, got.Name)
				xassert.Equal(t, arg.TaxRate, got.TaxRate)
				xassert.Equal(t, arg.Ticker, got.Ticker)
				xassert.Equal(t, arg.Shares.ValueOrZero(), got.Shares.ValueOrZero())
				xassert.Equal(t, arg.URL.ValueOrZero(), got.URL.ValueOrZero())
				xassert.Equal(t, arg.WarEligible.ValueOrZero(), got.WarEligible.ValueOrZero())
			}
		}
	})
	t.Run("can update existing full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
			Description:   optional.New("description"),
			FactionID:     optional.New(faction.ID),
			HomeStationID: optional.New(station.ID),
			ID:            42,
			MemberCount:   888,
			Name:          "name",
			Shares:        optional.New(int64(987)),
			TaxRate:       0.12,
			Ticker:        "ticker",
			URL:           optional.New("url"),
			WarEligible:   optional.New(false),
		}
		// when
		err := st.UpdateOrCreateEveCorporation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			got, err := st.GetEveCorporation(ctx, arg.ID)
			if assert.NoError(t, err) {
				xassert.Equal(t, alliance, got.Alliance.ValueOrZero())
				xassert.Equal(t, ceo, got.Ceo.ValueOrZero())
				xassert.Equal(t, arg.Description.ValueOrZero(), got.Description)
				xassert.Equal(t, faction, got.Faction.ValueOrZero())
				xassert.Equal(t, station, got.HomeStation.ValueOrZero())
				xassert.Equal(t, arg.ID, got.ID)
				xassert.Equal(t, arg.MemberCount, got.MemberCount)
				xassert.Equal(t, arg.Name, got.Name)
				xassert.Equal(t, arg.TaxRate, got.TaxRate)
				xassert.Equal(t, arg.Ticker, got.Ticker)
				xassert.Equal(t, arg.Shares.ValueOrZero(), got.Shares.ValueOrZero())
				xassert.Equal(t, arg.URL.ValueOrZero(), got.URL.ValueOrZero())
				xassert.Equal(t, arg.WarEligible.ValueOrZero(), got.WarEligible.ValueOrZero())
			}
		}
	})
	t.Run("can fetch by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateEveCorporation()
		// when
		c2, err := st.GetEveCorporation(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, c1.Name, c2.Name)
		}
	})
	t.Run("can fetch character by ID with all fields populated", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
			Description:   optional.New("description"),
			FactionID:     optional.New(faction.ID),
			HomeStationID: optional.New(station.ID),
			MemberCount:   42,
			Name:          "name",
			Shares:        optional.New[int64](888),
			TaxRate:       0.1,
			Ticker:        "ticker",
			URL:           optional.New("url"),
			WarEligible:   optional.New(true),
		})
		// when
		c2, err := st.GetEveCorporation(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, alliance, c2.Alliance.ValueOrZero())
			xassert.Equal(t, ceo, c2.Ceo.ValueOrZero())
			xassert.Equal(t, creator, c2.Creator.ValueOrZero())
			assert.True(t, c2.DateFounded.MustValue().Equal(founded))
			xassert.Equal(t, "description", c2.Description)
			xassert.Equal(t, faction, c2.Faction.ValueOrZero())
			xassert.Equal(t, station, c2.HomeStation.ValueOrZero())
			xassert.Equal(t, 42, c2.MemberCount)
			xassert.Equal(t, "name", c2.Name)
			xassert.Equal(t, 888, c2.Shares.MustValue())
			assert.InDelta(t, 0.1, c2.TaxRate, 0.01)
			xassert.Equal(t, "ticker", c2.Ticker)
			xassert.Equal(t, "url", c2.URL.ValueOrZero())
			xassert.Equal(t, c1.Name, c2.Name)
			assert.True(t, c2.WarEligible.ValueOrZero())
		}
	})
	t.Run("list corporation IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateEveCorporation()
		c2 := factory.CreateEveCorporation()
		// when
		got, err := st.ListEveCorporationIDs(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of(c1.ID, c2.ID)
			xassert.Equal(t, want, got)
		}
	})
	t.Run("can update name", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateEveCorporation()
		// when
		err := st.UpdateEveCorporationName(ctx, c1.ID, "Alpha")
		// then
		if assert.NoError(t, err) {
			// assert.False(t, created)
			r, err := st.GetEveCorporation(ctx, c1.ID)
			if assert.NoError(t, err) {
				xassert.Equal(t, "Alpha", r.Name)
			}
		}
	})
}
