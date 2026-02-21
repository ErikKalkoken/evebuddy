package eveuniverseservice

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateEveMarketPricesESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewTestService(st)
	ctx := context.Background()
	const (
		knownTypeID = 32772
		otherTypeID = 10001
	)
	t.Run("should create new objects from ESI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{
			ID: knownTypeID,
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"adjusted_price": 306988.09,
				"average_price":  306292.67,
				"type_id":        knownTypeID,
			}, {
				"adjusted_price": 123.45,
				"average_price":  456.78,
				"type_id":        otherTypeID,
			}}),
		)
		// when
		got, err := s.updateMarketPricesESI(ctx)
		// then
		require.NoError(t, err)
		want := set.Of[int64](knownTypeID)
		xassert.Equal(t, want, got)
		prices, err := st.ListEveMarketPrices(ctx)
		require.NoError(t, err)
		require.Len(t, prices, 2)
		for _, o := range prices {
			switch o.TypeID {
			case knownTypeID:
				xassert.Equal(t, 306988.09, o.AdjustedPrice.MustValue())
				xassert.Equal(t, 306292.67, o.AveragePrice.MustValue())
			case o.TypeID:
				xassert.Equal(t, 123.45, o.AdjustedPrice.MustValue())
				xassert.Equal(t, 456.78, o.AveragePrice.MustValue())
			}
		}
	})
	t.Run("should update existing objects from ESI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{
			ID: knownTypeID,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        knownTypeID,
			AdjustedPrice: optional.New(2.0),
			AveragePrice:  optional.New(3.0),
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"adjusted_price": 306988.09,
				"average_price":  306292.67,
				"type_id":        knownTypeID,
			}}),
		)
		// when
		got, err := s.updateMarketPricesESI(ctx)
		// then
		require.NoError(t, err)
		want := set.Of[int64](knownTypeID)
		xassert.Equal(t, want, got)
		o, err := st.GetEveMarketPrice(ctx, knownTypeID)
		require.NoError(t, err)
		xassert.Equal(t, 306988.09, o.AdjustedPrice.ValueOrZero())
		xassert.Equal(t, 306292.67, o.AveragePrice.ValueOrZero())
	})
	t.Run("should only report changes for known types", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{
			ID: knownTypeID,
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        knownTypeID,
			AdjustedPrice: optional.New(306988.09),
			AveragePrice:  optional.New(306292.67),
		})
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        otherTypeID,
			AdjustedPrice: optional.New(111.09),
			AveragePrice:  optional.New(222.67),
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"adjusted_price": 306988.09,
				"average_price":  306292.67,
				"type_id":        knownTypeID,
			}, {
				"adjusted_price": 123.45,
				"average_price":  456.78,
				"type_id":        otherTypeID,
			}}),
		)
		// when
		got, err := s.updateMarketPricesESI(ctx)
		// then
		require.NoError(t, err)
		want := set.Of[int64]()
		xassert.Equal(t, want, got)
	})

	t.Run("should remove obsolete prices", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			AdjustedPrice: optional.New(111.09),
			AveragePrice:  optional.New(222.67),
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"adjusted_price": 306988.09,
				"average_price":  306292.67,
				"type_id":        knownTypeID,
			}}),
		)
		// when
		_, err := s.updateMarketPricesESI(ctx)
		// then
		require.NoError(t, err)
		got, err := s.st.ListEveMarketPriceIDs(ctx)
		require.NoError(t, err)
		want := set.Of[int64](knownTypeID)
		xassert.Equal(t, want, got)
	})
}

func TestFormatDogmaValue(t *testing.T) {
	cases := []struct {
		name string
		args formatDogmaValueParams
		s    string
		v    int64
	}{
		{"AbsolutePercent", formatDogmaValueParams{value: 0.04, unitID: app.EveUnitAbsolutePercent}, "4%", 0},
		{"Acceleration", formatDogmaValueParams{value: 3, unitID: app.EveUnitAcceleration}, "3 m/s²", 0},
		{"AttributeID", formatDogmaValueParams{
			value:  3,
			unitID: app.EveUnitAttributeID,
			getDogmaAttribute: func(context.Context, int64) (*app.EveDogmaAttribute, error) {
				return &app.EveDogmaAttribute{
					DisplayName: optional.New("attribute"),
					IconID:      optional.New[int64](42),
				}, nil
			}},
			"attribute", 42},
		{"AttributePoints", formatDogmaValueParams{value: 42, unitID: app.EveUnitAttributePoints}, "42 points", 0},
		{"CapacitorUnits", formatDogmaValueParams{value: 123.45, unitID: app.EveUnitCapacitorUnits}, "123.5 GJ", 0},
		{"DroneBandwidth", formatDogmaValueParams{value: 10, unitID: app.EveUnitDroneBandwidth}, "10 Mbit/s", 0},
		{"Hitpoints", formatDogmaValueParams{value: 123, unitID: app.EveUnitHitpoints}, "123 HP", 0},
		{"InverseAbsolutePercent", formatDogmaValueParams{value: 0.04, unitID: app.EveUnitInverseAbsolutePercent}, "96%", 0},
		{"LengthSmall", formatDogmaValueParams{value: 123, unitID: app.EveUnitLength}, "123 m", 0},
		{"LengthBig", formatDogmaValueParams{value: 1234, unitID: app.EveUnitLength}, "1.234 km", 0},
		{"Level", formatDogmaValueParams{value: 5, unitID: app.EveUnitLevel}, "Level 5", 0},
		{"LightYear", formatDogmaValueParams{value: 1.23, unitID: app.EveUnitLightYear}, "1.2 LY", 0},
		{"Mass", formatDogmaValueParams{value: 42, unitID: app.EveUnitMass}, "42 kg", 0},
		{"MegaWatts", formatDogmaValueParams{value: 42, unitID: app.EveUnitMegaWatts}, "42 MW", 0},
		{"Millimeters", formatDogmaValueParams{value: 42, unitID: app.EveUnitMillimeters}, "42 mm", 0},
		{"Milliseconds", formatDogmaValueParams{value: 60_000, unitID: app.EveUnitMilliseconds}, "1 minute", 0},
		{"Multiplier", formatDogmaValueParams{value: 1.2345, unitID: app.EveUnitMultiplier}, "1.2345 x", 0},
		{"Percent", formatDogmaValueParams{value: 0.04, unitID: app.EveUnitPercentage}, "4%", 0},
		{"Teraflops", formatDogmaValueParams{value: 42, unitID: app.EveUnitTeraflops}, "42 tf", 0},
		{"Volume", formatDogmaValueParams{value: 10_001, unitID: app.EveUnitVolume}, "10,001 m3", 0},
		{"WarpSpeed", formatDogmaValueParams{value: 1.2345, unitID: app.EveUnitWarpSpeed}, "1.2345 AU/s", 0},
		{"TypeID", formatDogmaValueParams{value: 42, unitID: app.EveUnitTypeID, getType: func(ctx context.Context, i int64) (*app.EveType, error) {
			return &app.EveType{Name: "type", IconID: optional.New[int64](88)}, nil
		}}, "type", 88},
		{"Units", formatDogmaValueParams{value: 42, unitID: app.EveUnitUnits}, "42 units", 0},
		{"None", formatDogmaValueParams{value: 42, unitID: app.EveUnitNone}, "42", 0},
		{"Hardpoints", formatDogmaValueParams{value: 42, unitID: app.EveUnitHardpoints}, "42", 0},
		{"FittingSlots", formatDogmaValueParams{value: 42, unitID: app.EveUnitFittingSlots}, "42", 0},
		{"Slot", formatDogmaValueParams{value: 42, unitID: app.EveUnitSlot}, "42", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, v := formatDogmaValue(context.Background(), tc.args)
			xassert.Equal(t, tc.s, s)
			xassert.Equal(t, tc.v, v)
		})
	}
}
