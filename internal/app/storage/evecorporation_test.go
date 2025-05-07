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
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		arg := storage.UpdateOrCreateEveCorporationParams{
			ID:   1,
			Name: "Alpha",
		}
		// when
		err := r.UpdateOrCreateEveCorporation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			r, err := r.GetEveCorporation(ctx, arg.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, arg.Name, r.Name)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{MemberCount: 42})
		arg := storage.UpdateOrCreateEveCorporationParams{
			ID:          c1.ID,
			MemberCount: 99,
		}
		// when
		err := r.UpdateOrCreateEveCorporation(ctx, arg)
		// then
		if assert.NoError(t, err) {
			// assert.False(t, created)
			r, err := r.GetEveCorporation(ctx, arg.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 99, r.MemberCount)
			}
		}
	})
	t.Run("can fetch by ID with minimal fields populated only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateEveCorporation()
		// when
		c2, err := r.GetEveCorporation(ctx, c1.ID)
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
			AllianceID:    optional.From(alliance.ID),
			CeoID:         optional.From(ceo.ID),
			CreatorID:     optional.From(creator.ID),
			DateFounded:   optional.From(founded),
			Description:   "description",
			FactionID:     optional.From(faction.ID),
			HomeStationID: optional.From(station.ID),
			MemberCount:   42,
			Name:          "name",
			Shares:        optional.From[int64](888),
			TaxRate:       0.1,
			Ticker:        "ticker",
			URL:           "url",
			WarEligible:   true,
		})
		// when
		c2, err := r.GetEveCorporation(ctx, c1.ID)
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
		got, err := r.ListEveCorporationIDs(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of(c1.ID, c2.ID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}
