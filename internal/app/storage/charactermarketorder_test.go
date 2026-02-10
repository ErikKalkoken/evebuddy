package storage_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestCharacterMarketOrder(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		issued := time.Now().UTC()
		location := factory.CreateEveLocationStructure()
		itemType := factory.CreateEveType()
		region := factory.CreateEveRegion()
		owner := factory.CreateEveEntity(app.EveEntity{ID: c.ID})
		arg := storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID:   c.ID,
			Duration:      3,
			IsBuyOrder:    optional.New(true),
			IsCorporation: true,
			Issued:        issued,
			LocationID:    location.ID,
			OrderID:       42,
			OwnerID:       owner.ID,
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
				xassert.Equal(t, 3, o.Duration)
				assert.True(t, o.Escrow.IsEmpty())
				xassert.Equal(t, true, o.IsBuyOrder.ValueOrZero())
				xassert.Equal(t, true, o.IsCorporation)
				assert.True(t, issued.Equal(o.Issued), "got %q, wanted %q", issued, o.Issued)
				xassert.Equal(t, location.ID, o.Location.ID)
				xassert.Equal(t, location.Name, o.Location.Name.ValueOrZero())
				assert.True(t, o.MinVolume.IsEmpty())
				xassert.Equal(t, 123.45, o.Price)
				xassert.Equal(t, "station", o.Range)
				xassert.Equal(t, region.ID, o.Region.ID)
				xassert.Equal(t, region.Name, o.Region.Name)
				xassert.Equal(t, app.OrderOpen, o.State)
				xassert.Equal(t, itemType.ID, o.Type.ID)
				xassert.Equal(t, itemType.Name, o.Type.Name)
				xassert.Equal(t, 5, o.VolumeRemains)
				xassert.Equal(t, 10, o.VolumeTotal)
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		issued := time.Now().UTC()
		location := factory.CreateEveLocationStructure()
		itemType := factory.CreateEveType()
		region := factory.CreateEveRegion()
		owner := factory.CreateEveEntity(app.EveEntity{ID: c.ID})
		arg := storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID:   c.ID,
			Duration:      3,
			Escrow:        optional.New(234.56),
			IsBuyOrder:    optional.New(true),
			IsCorporation: true,
			Issued:        issued,
			LocationID:    location.ID,
			MinVolume:     optional.New[int64](3),
			OrderID:       42,
			OwnerID:       owner.ID,
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
				xassert.Equal(t, 234.56, o.Escrow.MustValue())
				xassert.Equal(t, 3, o.MinVolume.MustValue())
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		cmo := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			IsBuyOrder: optional.New(true),
		})
		remains := cmo.VolumeRemains - 1
		escrow := 1_000_000.12
		price := 123.45
		arg := storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID:   cmo.CharacterID,
			Duration:      cmo.Duration,
			Escrow:        optional.New(escrow),
			IsBuyOrder:    cmo.IsBuyOrder,
			IsCorporation: cmo.IsCorporation,
			Issued:        cmo.Issued,
			LocationID:    cmo.Location.ID,
			MinVolume:     cmo.MinVolume,
			OrderID:       cmo.OrderID,
			OwnerID:       cmo.Owner.ID,
			Price:         price,
			Range:         cmo.Range,
			RegionID:      cmo.Region.ID,
			State:         app.OrderExpired,
			TypeID:        cmo.Type.ID,
			VolumeRemains: remains,
			VolumeTotal:   cmo.VolumeTotal,
		}
		// when
		err := st.UpdateOrCreateCharacterMarketOrder(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCharacterMarketOrder(ctx, arg.CharacterID, arg.OrderID)
			if assert.NoError(t, err) {
				xassert.Equal(t, escrow, o.Escrow.ValueOrZero())
				xassert.Equal(t, price, o.Price)
				xassert.Equal(t, remains, o.VolumeRemains)
				xassert.Equal(t, app.OrderExpired, o.State)
			}
		}
	})
	t.Run("can list orders for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
		})
		o2 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterMarketOrder()
		// when
		s, err := st.ListCharacterMarketOrders(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := set.Of(o1.OrderID, o2.OrderID)
			got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterMarketOrder) int64 {
				return x.OrderID
			}))
			xassert.Equal2(t, want, got)
		}
	})
	t.Run("can list order IDs for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
		})
		o2 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterMarketOrder()
		// when
		got, err := st.ListCharacterMarketOrderIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := set.Of(o1.OrderID, o2.OrderID)
			xassert.Equal2(t, want, got)
		}
	})
	t.Run("can list all buy orders", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			IsBuyOrder: optional.New(true),
		})
		o2 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			IsBuyOrder: optional.New(true),
		})
		factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{})
		// when
		s, err := st.ListAllCharacterMarketOrders(ctx, true)
		// then
		if assert.NoError(t, err) {
			want := set.Of(o1.OrderID, o2.OrderID)
			got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterMarketOrder) int64 {
				return x.OrderID
			}))
			xassert.Equal2(t, want, got)
		}
	})
	t.Run("can list all sell orders", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{})
		o2 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{})
		factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			IsBuyOrder: optional.New(true),
		})
		// when
		s, err := st.ListAllCharacterMarketOrders(ctx, false)
		// then
		if assert.NoError(t, err) {
			want := set.Of(o1.OrderID, o2.OrderID)
			got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterMarketOrder) int64 {
				return x.OrderID
			}))
			xassert.Equal2(t, want, got)
		}
	})
	t.Run("can delete orders for a character by ID", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
		})
		o2 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
		})
		// when
		err := st.DeleteCharacterMarketOrders(ctx, c.ID, set.Of(o2.OrderID))
		// then
		if assert.NoError(t, err) {
			want := set.Of(o1.OrderID)
			got, err := st.ListCharacterMarketOrderIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				xassert.Equal2(t, want, got)
			}
		}
	})
	t.Run("can update order status", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			State:       app.OrderOpen,
		})
		// when
		err := st.UpdateCharacterMarketOrderState(ctx, storage.UpdateCharacterMarketOrderStateParams{
			CharacterID: c.ID,
			OrderIDs:    set.Of(o1.OrderID),
			State:       app.OrderUnknown,
		})
		// then
		failOnError(t, err)
		o2, err := st.GetCharacterMarketOrder(ctx, c.ID, o1.OrderID)
		failOnError(t, err)
		xassert.Equal(t, app.OrderUnknown, o2.State)
	})
}

func failOnError(t *testing.T, err error) {
	if err != nil {
		t.Fatal("Error occured", err)
	}
}
