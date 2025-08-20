package storage_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCharacterMarketOrder(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		issued := time.Now().UTC()
		location := factory.CreateEveLocationStructure()
		itemType := factory.CreateEveType()
		region := factory.CreateEveRegion()
		arg := storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID:   c.ID,
			Duration:      3,
			IsBuyOrder:    true,
			IsCorporation: true,
			Issued:        issued,
			LocationID:    location.ID,
			OrderID:       42,
			Price:         123.45,
			Range:         "station",
			RegionID:      region.ID,
			State:         app.OrderOpen,
			TypeID:        itemType.ID,
			VolumeRemains: 5,
			VolumeTotal:   10,
		}
		// when
		err := st.UpdateOrCreateCharacterMarketOrder(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCharacterMarketOrder(ctx, arg.CharacterID, arg.OrderID)
			if assert.NoError(t, err) {
				assert.EqualValues(t, 3, o.Duration)
				assert.True(t, o.Escrow.IsEmpty())
				assert.EqualValues(t, true, o.IsBuyOrder)
				assert.EqualValues(t, true, o.IsCorporation)
				assert.True(t, issued.Equal(o.Issued), "got %q, wanted %q", issued, o.Issued)
				assert.EqualValues(t, location.ID, o.LocationID)
				assert.True(t, o.MinVolume.IsEmpty())
				assert.EqualValues(t, 123.45, o.Price)
				assert.EqualValues(t, "station", o.Range)
				assert.EqualValues(t, region.ID, o.RegionID)
				assert.EqualValues(t, app.OrderOpen, o.State)
				assert.EqualValues(t, itemType.ID, o.TypeID)
				assert.EqualValues(t, 5, o.VolumeRemains)
				assert.EqualValues(t, 10, o.VolumeTotal)
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		issued := time.Now().UTC()
		location := factory.CreateEveLocationStructure()
		itemType := factory.CreateEveType()
		region := factory.CreateEveRegion()
		arg := storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID:   c.ID,
			Duration:      3,
			Escrow:        optional.New(234.56),
			IsBuyOrder:    true,
			IsCorporation: true,
			Issued:        issued,
			LocationID:    location.ID,
			MinVolume:     optional.New(3),
			OrderID:       42,
			Price:         123.45,
			Range:         "station",
			RegionID:      region.ID,
			State:         app.OrderOpen,
			TypeID:        itemType.ID,
			VolumeRemains: 5,
			VolumeTotal:   10,
		}
		// when
		err := st.UpdateOrCreateCharacterMarketOrder(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCharacterMarketOrder(ctx, arg.CharacterID, arg.OrderID)
			if assert.NoError(t, err) {
				assert.EqualValues(t, 234.56, o.Escrow.MustValue())
				assert.EqualValues(t, 3, o.MinVolume.MustValue())
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		cmo := factory.CreateCharacterMarketOrder()
		remains := cmo.VolumeRemains - 1
		arg := storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID:   cmo.CharacterID,
			Duration:      cmo.Duration,
			Issued:        cmo.Issued,
			LocationID:    cmo.LocationID,
			OrderID:       cmo.OrderID,
			RegionID:      cmo.RegionID,
			State:         app.OrderExpired,
			TypeID:        cmo.TypeID,
			VolumeRemains: remains,
			VolumeTotal:   cmo.VolumeTotal,
		}
		// when
		err := st.UpdateOrCreateCharacterMarketOrder(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCharacterMarketOrder(ctx, arg.CharacterID, arg.OrderID)
			if assert.NoError(t, err) {
				assert.EqualValues(t, app.OrderExpired, o.State)
				assert.EqualValues(t, remains, o.VolumeRemains)
			}
		}
	})
	t.Run("can list jobs for a character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		j1 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
		})
		j2 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterMarketOrder()
		// when
		s, err := st.ListCharacterMarketOrders(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := set.Of(j1.OrderID, j2.OrderID)
			got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterMarketOrder) int64 {
				return x.OrderID
			}))
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})

}
