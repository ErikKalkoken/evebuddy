package storage_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		require.NoError(t, err)
		o, err := st.GetCharacterMarketOrder(ctx, arg.CharacterID, arg.OrderID)
		require.NoError(t, err)
		xassert.Equal(t, 3, o.Duration)
		assert.True(t, o.Escrow.IsEmpty())
		xassert.EqualOptional(t, true, o.IsBuyOrder)
		xassert.Equal(t, true, o.IsCorporation)
		assert.True(t, issued.Equal(o.Issued), "got %q, wanted %q", issued, o.Issued)
		xassert.Equal(t, location.ID, o.Location.ID)
		xassert.EqualOptional(t, location.Name, o.Location.Name)
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
		require.NoError(t, err)
		o, err := st.GetCharacterMarketOrder(ctx, arg.CharacterID, arg.OrderID)
		require.NoError(t, err)
		xassert.EqualOptional(t, 234.56, o.Escrow)
		xassert.EqualOptional(t, 3, o.MinVolume)
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
		require.NoError(t, err)
		o, err := st.GetCharacterMarketOrder(ctx, arg.CharacterID, arg.OrderID)
		require.NoError(t, err)
		xassert.EqualOptional(t, escrow, o.Escrow)
		xassert.Equal(t, price, o.Price)
		xassert.Equal(t, remains, o.VolumeRemains)
		xassert.Equal(t, app.OrderExpired, o.State)
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
		require.NoError(t, err)
		want := set.Of(o1.OrderID, o2.OrderID)
		got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterMarketOrder) int64 {
			return x.OrderID
		}))
		xassert.Equal(t, want, got)
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
		require.NoError(t, err)
		want := set.Of(o1.OrderID, o2.OrderID)
		xassert.Equal(t, want, got)
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
		require.NoError(t, err)
		want := set.Of(o1.OrderID, o2.OrderID)
		got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterMarketOrder) int64 {
			return x.OrderID
		}))
		xassert.Equal(t, want, got)
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
		require.NoError(t, err)
		want := set.Of(o1.OrderID, o2.OrderID)
		got := set.Collect(xiter.Map(slices.Values(s), func(x *app.CharacterMarketOrder) int64 {
			return x.OrderID
		}))
		xassert.Equal(t, want, got)
	})
	t.Run("returns not found error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.GetCharacterMarketOrder(t.Context(), 1, 2)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestCalculateCharacterOrderItemsValue(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()

	const characterID = 42

	cases := []struct {
		name          string
		isBuyOrder    bool
		isCorporation bool
		state         app.MarketOrderState
		price         float64
		want          float64
	}{
		{"should include open sell order", false, false, app.OrderOpen, 10.0, 10.0},
		{"should ignore expired sell order", false, false, app.OrderExpired, 10.0, 0},
		{"should ignore buy orders", true, false, app.OrderOpen, 0, 0},
		{"should ignore cancelled orders", false, false, app.OrderCancelled, 0, 0},
		{"should ignore sell order placed for corporation", false, true, app.OrderOpen, 10.0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
				ID: characterID,
			})
			c := factory.CreateCharacter(storage.CreateCharacterParams{
				ID: ec.ID,
			})
			o := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
				CharacterID:   c.ID,
				IsBuyOrder:    optional.New(tc.isBuyOrder),
				IsCorporation: tc.isCorporation,
				State:         tc.state,
				VolumeRemains: 1,
			})
			factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
				TypeID:       o.Type.ID,
				AveragePrice: optional.New(tc.price),
			})

			// when
			got, err := st.CalculateCharacterOrderItemsValue(t.Context(), c.ID)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("can calculate items value for multiple orders", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o1 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID:   c.ID,
			IsBuyOrder:    optional.New(false),
			State:         app.OrderOpen,
			VolumeRemains: 1,
		})
		o2 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID:   c.ID,
			IsBuyOrder:    optional.New(false),
			State:         app.OrderOpen,
			VolumeRemains: 2,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       o1.Type.ID,
			AveragePrice: optional.New(100.1),
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       o2.Type.ID,
			AveragePrice: optional.New(200.2),
		})

		// when
		got, err := st.CalculateCharacterOrderItemsValue(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		assert.Equal(t, 500.5, got)
	})

	t.Run("should ignore blueprints", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		blueprintCategory := factory.CreateEveCategory(storage.CreateEveCategoryParams{
			ID:   app.EveCategoryBlueprint,
			Name: "Blueprint",
		})
		blueprintGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{
			CategoryID: blueprintCategory.ID,
		})
		blueprintType := factory.CreateEveType(storage.CreateEveTypeParams{
			GroupID: blueprintGroup.ID,
		})
		factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID:   c.ID,
			IsBuyOrder:    optional.New(false),
			State:         app.OrderOpen,
			TypeID:        blueprintType.ID,
			VolumeRemains: 1,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:       blueprintType.ID,
			AveragePrice: optional.New(66.6),
		})

		// when
		got, err := st.CalculateCharacterOrderItemsValue(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0.0, got)
	})
}

func TestCalcOrdersEscrow(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	cases := []struct {
		name          string
		isBuyOrder    optional.Optional[bool]
		isCorporation bool
		state         app.MarketOrderState
		escrow        optional.Optional[float64]
		want          float64
	}{
		{"should include open buy order with escrow", optional.New(true), false, app.OrderOpen, optional.New(12.3), 12.3},
		{"should not include buy order without escrow", optional.New(true), false, app.OrderOpen, optional.Optional[float64]{}, 0.0},
		{"should not include sell order", optional.New(false), false, app.OrderOpen, optional.New(12.3), 0.0},
		{"should not include expired buy order", optional.New(true), false, app.OrderExpired, optional.New(12.3), 0.0},
		{"should not include buy order placed for corporation", optional.New(true), true, app.OrderOpen, optional.New(12.3), 0.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			c := factory.CreateCharacter()
			factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
				CharacterID:   c.ID,
				IsBuyOrder:    tc.isBuyOrder,
				IsCorporation: tc.isCorporation,
				State:         tc.state,
				Escrow:        tc.escrow,
			})

			// when
			got, err := st.CalculateCharacterOrdersEscrow(t.Context(), c.ID)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("should sum buy orders owner by character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			IsBuyOrder:  optional.New(true),
			Escrow:      optional.New(12.3),
		})
		factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			IsBuyOrder:  optional.New(true),
			Escrow:      optional.New(10.1),
		})
		factory.CreateCharacterMarketOrder() // should be ignored

		// when
		got, err := st.CalculateCharacterOrdersEscrow(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		assert.Equal(t, 22.4, got)
	})
}

func TestReplaceCharacterMarketOrders(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("should create new orders", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		owner := factory.CreateEveEntity(app.EveEntity{ID: c.ID})
		location := factory.CreateEveLocationStructure()
		region := factory.CreateEveRegion()
		typ := factory.CreateEveType()
		issued := time.Now().UTC()
		arg := storage.CharacterMarketOrderParams{
			Duration:      3,
			Escrow:        optional.New(234.56),
			IsBuyOrder:    optional.New(true),
			IsCorporation: true,
			Issued:        issued,
			LocationID:    location.ID,
			MinVolume:     optional.New[int64](2),
			OrderID:       42,
			OwnerID:       owner.ID,
			Price:         123.45,
			Range:         "station",
			RegionID:      region.ID,
			State:         app.OrderOpen,
			TypeID:        typ.ID,
			VolumeRemains: 5,
			VolumeTotal:   10,
		}
		// when
		changed, err := st.ReplaceCharacterMarketOrders(t.Context(), c.ID, 0, []storage.CharacterMarketOrderParams{arg})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o, err := st.GetCharacterMarketOrder(t.Context(), c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, 3, o.Duration)
		xassert.EqualOptional(t, 234.56, o.Escrow)
		xassert.EqualOptional(t, true, o.IsBuyOrder)
		xassert.Equal(t, true, o.IsCorporation)
		assert.True(t, issued.Equal(o.Issued), "got %q, wanted %q", o.Issued, issued)
		xassert.Equal(t, location.ID, o.Location.ID)
		xassert.EqualOptional(t, 2, o.MinVolume)
		xassert.Equal(t, owner.ID, o.Owner.ID)
		xassert.Equal(t, 123.45, o.Price)
		xassert.Equal(t, "station", o.Range)
		xassert.Equal(t, region.ID, o.Region.ID)
		xassert.Equal(t, app.OrderOpen, o.State)
		xassert.Equal(t, typ.ID, o.Type.ID)
		xassert.Equal(t, 5, o.VolumeRemains)
		xassert.Equal(t, 10, o.VolumeTotal)
	})

	t.Run("should report no change when orders are unchanged", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		owner := factory.CreateEveEntity(app.EveEntity{ID: c.ID})
		location := factory.CreateEveLocationStructure()
		region := factory.CreateEveRegion()
		typ := factory.CreateEveType()
		args := []storage.CharacterMarketOrderParams{
			{
				Duration:      3,
				Escrow:        optional.New(234.56),
				IsBuyOrder:    optional.New(true),
				Issued:        time.Now().In(time.FixedZone("test", 3600)), // differs from UTC after the DB round trip
				LocationID:    location.ID,
				OrderID:       42,
				OwnerID:       owner.ID,
				Price:         123.45,
				RegionID:      region.ID,
				State:         app.OrderOpen,
				TypeID:        typ.ID,
				VolumeRemains: 5,
				VolumeTotal:   10,
			},
			{
				Duration:      3,
				Issued:        time.Now().Add(-48 * time.Hour),
				LocationID:    location.ID,
				OrderID:       43,
				OwnerID:       owner.ID,
				Price:         99.5,
				RegionID:      region.ID,
				State:         app.OrderExpired,
				TypeID:        typ.ID,
				VolumeRemains: 1,
				VolumeTotal:   10,
			},
		}
		_, err := st.ReplaceCharacterMarketOrders(t.Context(), c.ID, 0, args)
		require.NoError(t, err)
		// when
		changed, err := st.ReplaceCharacterMarketOrders(t.Context(), c.ID, 0, args)
		// then
		require.NoError(t, err)
		assert.False(t, changed)
	})

	t.Run("should store an order when a changeable column has changed", func(t *testing.T) {
		cases := []struct {
			name   string
			change func(arg *storage.CharacterMarketOrderParams)
		}{
			{"escrow", func(arg *storage.CharacterMarketOrderParams) {
				arg.Escrow = optional.New(1_000.5)
			}},
			{"escrow removed", func(arg *storage.CharacterMarketOrderParams) {
				arg.Escrow = optional.Optional[float64]{}
			}},
			{"issued", func(arg *storage.CharacterMarketOrderParams) {
				arg.Issued = arg.Issued.Add(time.Hour) // editing an order resets issued
			}},
			{"price", func(arg *storage.CharacterMarketOrderParams) {
				arg.Price = 200.25
			}},
			{"state", func(arg *storage.CharacterMarketOrderParams) {
				arg.State = app.OrderCancelled
			}},
			{"volume remains", func(arg *storage.CharacterMarketOrderParams) {
				arg.VolumeRemains = 4
			}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// given
				testutil.MustTruncateTables(db)
				c := factory.CreateCharacter()
				owner := factory.CreateEveEntity(app.EveEntity{ID: c.ID})
				location := factory.CreateEveLocationStructure()
				region := factory.CreateEveRegion()
				typ := factory.CreateEveType()
				arg := storage.CharacterMarketOrderParams{
					Duration:      3,
					Escrow:        optional.New(234.56),
					IsBuyOrder:    optional.New(true),
					Issued:        time.Now().UTC(),
					LocationID:    location.ID,
					OrderID:       42,
					OwnerID:       owner.ID,
					Price:         123.45,
					RegionID:      region.ID,
					State:         app.OrderOpen,
					TypeID:        typ.ID,
					VolumeRemains: 5,
					VolumeTotal:   10,
				}
				_, err := st.ReplaceCharacterMarketOrders(t.Context(), c.ID, 0, []storage.CharacterMarketOrderParams{arg})
				require.NoError(t, err)
				tc.change(&arg)
				// when
				changed, err := st.ReplaceCharacterMarketOrders(t.Context(), c.ID, 0, []storage.CharacterMarketOrderParams{arg})
				// then
				require.NoError(t, err)
				assert.True(t, changed)
				o, err := st.GetCharacterMarketOrder(t.Context(), c.ID, 42)
				require.NoError(t, err)
				xassert.Equal(t, arg.Escrow, o.Escrow)
				assert.True(t, arg.Issued.Equal(o.Issued), "got %q, wanted %q", o.Issued, arg.Issued)
				xassert.Equal(t, arg.Price, o.Price)
				xassert.Equal(t, arg.State, o.State)
				xassert.Equal(t, arg.VolumeRemains, o.VolumeRemains)
			})
		}
	})

	t.Run("should ignore changes to columns which can not change", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		owner := factory.CreateEveEntity(app.EveEntity{ID: c.ID})
		location := factory.CreateEveLocationStructure()
		region := factory.CreateEveRegion()
		typ := factory.CreateEveType()
		arg := storage.CharacterMarketOrderParams{
			Duration:      3,
			Issued:        time.Now().UTC(),
			LocationID:    location.ID,
			OrderID:       42,
			OwnerID:       owner.ID,
			Price:         123.45,
			Range:         "station",
			RegionID:      region.ID,
			State:         app.OrderOpen,
			TypeID:        typ.ID,
			VolumeRemains: 5,
			VolumeTotal:   10,
		}
		_, err := st.ReplaceCharacterMarketOrders(t.Context(), c.ID, 0, []storage.CharacterMarketOrderParams{arg})
		require.NoError(t, err)
		arg.Range = "region"
		// when
		changed, err := st.ReplaceCharacterMarketOrders(t.Context(), c.ID, 0, []storage.CharacterMarketOrderParams{arg})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		o, err := st.GetCharacterMarketOrder(t.Context(), c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, "station", o.Range)
	})

	t.Run("should mark open orders missing in args as unknown", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		open := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			State:       app.OrderOpen,
		})
		expired := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			State:       app.OrderExpired,
		})
		// when
		changed, err := st.ReplaceCharacterMarketOrders(t.Context(), c.ID, 0, nil)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		o1, err := st.GetCharacterMarketOrder(t.Context(), c.ID, open.OrderID)
		require.NoError(t, err)
		xassert.Equal(t, app.OrderUnknown, o1.State)
		o2, err := st.GetCharacterMarketOrder(t.Context(), c.ID, expired.OrderID)
		require.NoError(t, err)
		xassert.Equal(t, app.OrderExpired, o2.State)
	})

	t.Run("should delete closed orders older than retention", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		old := time.Now().UTC().Add(-48 * time.Hour)
		recent := time.Now().UTC().Add(-time.Hour)
		openOld := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			Issued:      old,
			State:       app.OrderOpen,
		})
		expiredRecent := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			Issued:      recent,
			State:       app.OrderExpired,
		})
		factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			Issued:      old,
			State:       app.OrderExpired,
		})
		factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			Issued:      old,
			State:       app.OrderOpen,
		})
		cancelledOldInArgs := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			Issued:      old,
			State:       app.OrderOpen,
		})
		args := []storage.CharacterMarketOrderParams{
			{
				Duration:      openOld.Duration,
				Issued:        openOld.Issued,
				LocationID:    openOld.Location.ID,
				OrderID:       openOld.OrderID,
				OwnerID:       openOld.Owner.ID,
				Price:         openOld.Price,
				RegionID:      openOld.Region.ID,
				State:         app.OrderOpen,
				TypeID:        openOld.Type.ID,
				VolumeRemains: openOld.VolumeRemains,
				VolumeTotal:   openOld.VolumeTotal,
			},
			{
				Duration:      cancelledOldInArgs.Duration,
				Issued:        cancelledOldInArgs.Issued,
				LocationID:    cancelledOldInArgs.Location.ID,
				OrderID:       cancelledOldInArgs.OrderID,
				OwnerID:       cancelledOldInArgs.Owner.ID,
				Price:         cancelledOldInArgs.Price,
				RegionID:      cancelledOldInArgs.Region.ID,
				State:         app.OrderCancelled,
				TypeID:        cancelledOldInArgs.Type.ID,
				VolumeRemains: cancelledOldInArgs.VolumeRemains,
				VolumeTotal:   cancelledOldInArgs.VolumeTotal,
			},
			{
				Duration:      3,
				Issued:        old,
				LocationID:    openOld.Location.ID,
				OrderID:       openOld.OrderID + 100,
				OwnerID:       openOld.Owner.ID,
				Price:         1.5,
				RegionID:      openOld.Region.ID,
				State:         app.OrderExpired,
				TypeID:        openOld.Type.ID,
				VolumeRemains: 1,
				VolumeTotal:   2,
			},
		}
		// when
		changed, err := st.ReplaceCharacterMarketOrders(t.Context(), c.ID, 24*time.Hour, args)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCharacterMarketOrderIDs(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(openOld.OrderID, expiredRecent.OrderID), got)
	})

	t.Run("should keep old closed orders when retention is zero", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		o := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			Issued:      time.Now().UTC().Add(-365 * 24 * time.Hour),
			State:       app.OrderExpired,
		})
		// when
		changed, err := st.ReplaceCharacterMarketOrders(t.Context(), c.ID, 0, nil)
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		got, err := st.ListCharacterMarketOrderIDs(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(o.OrderID), got)
	})

	t.Run("should report no change when there are no orders", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		// when
		changed, err := st.ReplaceCharacterMarketOrders(t.Context(), c.ID, 24*time.Hour, nil)
		// then
		require.NoError(t, err)
		assert.False(t, changed)
	})

	t.Run("should not touch orders of other characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacter()
		c2 := factory.CreateCharacter()
		other := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c2.ID,
			Issued:      time.Now().UTC().Add(-48 * time.Hour),
			State:       app.OrderOpen,
		})
		// when
		changed, err := st.ReplaceCharacterMarketOrders(t.Context(), c1.ID, 24*time.Hour, nil)
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		o, err := st.GetCharacterMarketOrder(t.Context(), c2.ID, other.OrderID)
		require.NoError(t, err)
		xassert.Equal(t, app.OrderOpen, o.State)
	})

	t.Run("should keep orders unchanged when replace fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		existing := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			State:       app.OrderOpen,
		})
		arg := storage.CharacterMarketOrderParams{
			Duration:      3,
			Issued:        time.Now().UTC(),
			LocationID:    existing.Location.ID,
			OrderID:       existing.OrderID + 1,
			OwnerID:       existing.Owner.ID,
			Price:         1.5,
			RegionID:      existing.Region.ID,
			State:         app.OrderOpen,
			TypeID:        42, // has no eve type
			VolumeRemains: 1,
			VolumeTotal:   2,
		}
		// when
		_, err := st.ReplaceCharacterMarketOrders(t.Context(), c.ID, 0, []storage.CharacterMarketOrderParams{arg})
		// then
		require.Error(t, err)
		got, err := st.ListCharacterMarketOrderIDs(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(existing.OrderID), got)
		o, err := st.GetCharacterMarketOrder(t.Context(), c.ID, existing.OrderID)
		require.NoError(t, err)
		xassert.Equal(t, app.OrderOpen, o.State) // not marked as unknown
	})

	t.Run("should return error when args are invalid", func(t *testing.T) {
		valid := storage.CharacterMarketOrderParams{
			Duration:   3,
			Issued:     time.Now(),
			LocationID: 1,
			OrderID:    2,
			OwnerID:    3,
			RegionID:   4,
			TypeID:     5,
		}
		cases := []struct {
			name        string
			characterID int64
			retention   time.Duration
			change      func(arg *storage.CharacterMarketOrderParams)
		}{
			{"zero character ID", 0, 0, func(_ *storage.CharacterMarketOrderParams) {}},
			{"negative retention", 1, -time.Hour, func(_ *storage.CharacterMarketOrderParams) {}},
			{"zero duration", 1, 0, func(arg *storage.CharacterMarketOrderParams) { arg.Duration = 0 }},
			{"zero issued", 1, 0, func(arg *storage.CharacterMarketOrderParams) { arg.Issued = time.Time{} }},
			{"zero location ID", 1, 0, func(arg *storage.CharacterMarketOrderParams) { arg.LocationID = 0 }},
			{"zero order ID", 1, 0, func(arg *storage.CharacterMarketOrderParams) { arg.OrderID = 0 }},
			{"zero owner ID", 1, 0, func(arg *storage.CharacterMarketOrderParams) { arg.OwnerID = 0 }},
			{"zero region ID", 1, 0, func(arg *storage.CharacterMarketOrderParams) { arg.RegionID = 0 }},
			{"zero type ID", 1, 0, func(arg *storage.CharacterMarketOrderParams) { arg.TypeID = 0 }},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// given
				arg := valid
				tc.change(&arg)
				// when
				_, err := st.ReplaceCharacterMarketOrders(t.Context(), tc.characterID, tc.retention, []storage.CharacterMarketOrderParams{arg})
				// then
				assert.ErrorIs(t, err, app.ErrInvalid)
			})
		}
	})
}
