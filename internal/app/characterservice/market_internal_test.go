package characterservice

import (
	"context"
	"fmt"
	"maps"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateCharacterMarketOrdersESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can create new order from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntity(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		itemType1 := factory.CreateEveType()
		itemType2 := factory.CreateEveType()
		location1 := factory.CreateEveLocationStation()
		location2 := factory.CreateEveLocationStation()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/orders/history", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"duration":       9,
				"is_buy_order":   false,
				"is_corporation": false,
				"issued":         "2019-07-24T14:15:22Z",
				"location_id":    location2.ID,
				"order_id":       12,
				"price":          0.45,
				"range":          "station",
				"region_id":      location2.SolarSystem.MustValue().Constellation.Region.ID,
				"state":          "expired",
				"type_id":        itemType2.ID,
				"volume_remain":  1,
				"volume_total":   100,
			}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/orders", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"duration":       3,
				"is_buy_order":   true,
				"is_corporation": true,
				"issued":         "2019-08-24T14:15:22Z",
				"location_id":    location1.ID,
				"order_id":       42,
				"price":          123.45,
				"range":          "1",
				"region_id":      location1.SolarSystem.MustValue().Constellation.Region.ID,
				"type_id":        itemType1.ID,
				"volume_remain":  5,
				"volume_total":   10,
			}}),
		)
		// when
		changed, err := s.updateMarketOrdersESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterMarketOrders,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			oo, err := st.ListCharacterMarketOrders(ctx, c.ID)
			if assert.NoError(t, err) {
				m := make(map[int64]*app.CharacterMarketOrder)
				for _, o := range oo {
					m[o.OrderID] = o
				}
				want := set.Of[int64](12, 42)
				got := set.Collect(maps.Keys(m))
				xassert.Equal(t, want, got)
				o := m[42]
				issued := time.Date(2019, 8, 24, 14, 15, 22, 0, time.UTC)
				xassert.Equal(t, 3, o.Duration)
				assert.True(t, o.Escrow.IsEmpty())
				xassert.Equal(t, true, o.IsBuyOrder.ValueOrZero())
				xassert.Equal(t, true, o.IsCorporation)
				assert.True(t, issued.Equal(o.Issued), "got %q, wanted %q", issued, o.Issued)
				xassert.Equal(t, location1.ID, o.Location.ID)
				assert.True(t, o.MinVolume.IsEmpty())
				xassert.Equal(t, 123.45, o.Price)
				xassert.Equal(t, "1", o.Range)
				xassert.Equal(t, location1.SolarSystem.MustValue().Constellation.Region.ID, o.Region.ID)
				xassert.Equal(t, app.OrderOpen, o.State)
				xassert.Equal(t, itemType1.ID, o.Type.ID)
				xassert.Equal(t, 5, o.VolumeRemains)
				xassert.Equal(t, 10, o.VolumeTotal)
			}
		}
	})
	t.Run("can update existing orders", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		o1 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			IsBuyOrder:  optional.New(true),
		})
		remain := o1.VolumeRemains - 1
		price := 1234.56
		escrow := 1_000_000.12
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/orders/history", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"escrow":         escrow,
				"duration":       o1.Duration,
				"is_buy_order":   o1.IsBuyOrder,
				"is_corporation": o1.IsCorporation,
				"issued":         o1.Issued.Format(time.RFC3339),
				"location_id":    o1.Location.ID,
				"order_id":       o1.OrderID,
				"price":          price,
				"range":          o1.Range,
				"region_id":      o1.Region.ID,
				"state":          "expired",
				"type_id":        o1.Type.ID,
				"volume_remain":  remain,
				"volume_total":   o1.VolumeTotal,
			}}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/orders", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
		)
		// when
		changed, err := s.updateMarketOrdersESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterMarketOrders,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			oo, err := st.ListCharacterMarketOrders(ctx, c.ID)
			if assert.NoError(t, err) {
				if assert.Len(t, oo, 1) {
					o2 := oo[0]
					assert.InDelta(t, escrow, o2.Escrow.ValueOrZero(), 0.01)
					xassert.Equal(t, remain, o2.VolumeRemains)
					assert.InDelta(t, price, o2.Price, 0.01)
					xassert.Equal(t, app.OrderExpired, o2.State)
				}
			}
		}
	})
	t.Run("should mark orphaned orders with state unknown", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		o1 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			State:       app.OrderOpen,
		})
		o2 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			State:       app.OrderOpen,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/orders/history", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/orders", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"duration":       o1.Duration,
				"is_buy_order":   o1.IsBuyOrder,
				"is_corporation": o1.IsCorporation,
				"issued":         o1.Issued.Format(time.RFC3339),
				"location_id":    o1.Location.ID,
				"order_id":       o1.OrderID,
				"price":          o1.Price,
				"range":          o1.Range,
				"region_id":      o1.Region.ID,
				"type_id":        o1.Type.ID,
				"volume_remain":  o1.VolumeRemains,
				"volume_total":   o1.VolumeTotal,
			}}),
		)
		// when
		changed, err := s.updateMarketOrdersESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterMarketOrders,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o2a, err := st.GetCharacterMarketOrder(ctx, o2.CharacterID, o2.OrderID)
			if assert.NoError(t, err) {
				xassert.Equal(t, app.OrderUnknown, o2a.State)
			}
		}
	})
	t.Run("should delete stale orders", func(t *testing.T) {
		// given
		settings := &testutil.SettingsStub{MarketOrderRetentionDaysDefault: 90}
		s2 := NewFake(Params{Settings: settings, Storage: st})
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		o1 := factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			State:       app.OrderOpen,
		})
		factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID: c.ID,
			State:       app.OrderOpen,
			Issued:      time.Now().Add(-time.Hour * 24 * 91),
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/orders/history", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/orders", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"duration":       o1.Duration,
				"is_buy_order":   o1.IsBuyOrder,
				"is_corporation": o1.IsCorporation,
				"issued":         o1.Issued.Format(time.RFC3339),
				"location_id":    o1.Location.ID,
				"order_id":       o1.OrderID,
				"price":          o1.Price,
				"range":          o1.Range,
				"region_id":      o1.Region.ID,
				"type_id":        o1.Type.ID,
				"volume_remain":  o1.VolumeRemains,
				"volume_total":   o1.VolumeTotal,
			}}),
		)
		// when
		changed, err := s2.updateMarketOrdersESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterMarketOrders,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			got, err := st.ListCharacterMarketOrderIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				want := set.Of(o1.OrderID)
				xassert.Equal(t, want, got)
			}
		}
	})
	t.Run("should ignore invalid orders", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateEveEntity(app.EveEntity{ID: c.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		itemType := factory.CreateEveType()
		location := factory.CreateEveLocationStation()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/orders/history", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"duration":       9,
					"is_buy_order":   false,
					"is_corporation": false,
					"issued":         "2019-07-24T14:15:22Z",
					"location_id":    location.ID,
					"order_id":       12,
					"price":          0.45,
					"range":          "station",
					"region_id":      location.SolarSystem.MustValue().Constellation.Region.ID,
					"state":          "expired",
					"type_id":        itemType.ID,
					"volume_remain":  1,
					"volume_total":   100,
				},
				{
					"duration":       0, // invalid duration
					"is_buy_order":   false,
					"is_corporation": false,
					"issued":         "2019-07-24T14:15:22Z",
					"location_id":    location.ID,
					"order_id":       13,
					"price":          0.45,
					"range":          "station",
					"region_id":      location.SolarSystem.MustValue().Constellation.Region.ID,
					"state":          "expired",
					"type_id":        itemType.ID,
					"volume_remain":  1,
					"volume_total":   100,
				},
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/orders", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{}),
		)
		// when
		changed, err := s.updateMarketOrdersESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterMarketOrders,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			got, err := st.ListCharacterMarketOrderIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				want := set.Of[int64](12)
				xassert.Equal(t, want, got)
			}
		}
	})
}

func TestUpdateOrdersEscrow(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})

	cases := []struct {
		name             string
		firstIsBuyOrder  optional.Optional[bool]
		firstEscrow      optional.Optional[float64]
		secondIsBuyOrder optional.Optional[bool]
		secondEscrow     optional.Optional[float64]
		want             optional.Optional[float64]
	}{
		{"normal", optional.New(true), optional.New(12.3), optional.New(true), optional.New(10.0), optional.New(22.3)},
		{"buy order without escrow", optional.New(true), optional.Optional[float64]{}, optional.New(true), optional.New(10.0), optional.New(10.0)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			c := factory.CreateCharacter()
			factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
				CharacterID: c.ID,
				IsBuyOrder:  tc.firstIsBuyOrder,
				Escrow:      tc.firstEscrow,
			})
			factory.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
				CharacterID: c.ID,
				IsBuyOrder:  tc.secondIsBuyOrder,
				Escrow:      tc.secondEscrow,
			})
			factory.CreateCharacterMarketOrder() // should be ignored

			// when
			err := s.updateOrdersEscrow(t.Context(), c.ID)

			// then
			require.NoError(t, err)
			c2, err := st.GetCharacter(t.Context(), c.ID)
			require.NoError(t, err)
			got := c2.OrdersEscrow
			assert.Equal(t, tc.want, got)
		})
	}
}
