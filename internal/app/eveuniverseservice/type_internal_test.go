package eveuniverseservice

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

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
	t.Run("should create new objects from ESI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"adjusted_price": 306988.09,
					"average_price":  306292.67,
					"type_id":        32772,
				},
			}))
		// when
		got, err := s.updateMarketPricesESI(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of[int64](32772)
			xassert.Equal2(t, want, got)
			o, err := st.GetEveMarketPrice(ctx, 32772)
			if assert.NoError(t, err) {
				xassert.Equal(t, 306988.09, o.AdjustedPrice.ValueOrZero())
				xassert.Equal(t, 306292.67, o.AveragePrice.ValueOrZero())
			}
		}
	})
	t.Run("should update existing objects from ESI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        32772,
			AdjustedPrice: optional.New(2.0),
			AveragePrice:  optional.New(3.0),
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"adjusted_price": 306988.09,
					"average_price":  306292.67,
					"type_id":        32772,
				},
			}))
		// when
		got, err := s.updateMarketPricesESI(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of[int64](32772)
			xassert.Equal2(t, want, got)
			o, err := st.GetEveMarketPrice(ctx, 32772)
			if assert.NoError(t, err) {
				xassert.Equal(t, 306988.09, o.AdjustedPrice.ValueOrZero())
				xassert.Equal(t, 306292.67, o.AveragePrice.ValueOrZero())
			}
		}
	})
	t.Run("should detect when object has not changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
			TypeID:        32772,
			AdjustedPrice: optional.New(306988.09),
			AveragePrice:  optional.New(306292.67),
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/markets/prices",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"adjusted_price": 306988.09,
					"average_price":  306292.67,
					"type_id":        32772,
				},
			}))
		// when
		got, err := s.updateMarketPricesESI(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of[int64]()
			xassert.Equal2(t, want, got)
		}
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
		{"LengthBig", formatDogmaValueParams{value: 1234, unitID: app.EveUnitLength}, "1.23 km", 0},
		{"Level", formatDogmaValueParams{value: 5, unitID: app.EveUnitLevel}, "Level 5", 0},
		{"LightYear", formatDogmaValueParams{value: 1.23, unitID: app.EveUnitLightYear}, "1.2 LY", 0},
		{"Mass", formatDogmaValueParams{value: 42, unitID: app.EveUnitMass}, "42 kg", 0},
		{"MegaWatts", formatDogmaValueParams{value: 42, unitID: app.EveUnitMegaWatts}, "42 MW", 0},
		{"Millimeters", formatDogmaValueParams{value: 42, unitID: app.EveUnitMillimeters}, "42 mm", 0},
		{"Milliseconds", formatDogmaValueParams{value: 60_000, unitID: app.EveUnitMilliseconds}, "1 minute", 0},
		{"Multiplier", formatDogmaValueParams{value: 1.2345, unitID: app.EveUnitMultiplier}, "1.234 x", 0},
		{"Percent", formatDogmaValueParams{value: 0.04, unitID: app.EveUnitPercentage}, "4%", 0},
		{"Teraflops", formatDogmaValueParams{value: 42, unitID: app.EveUnitTeraflops}, "42 tf", 0},
		{"Volume", formatDogmaValueParams{value: 10_001, unitID: app.EveUnitVolume}, "10,001 m3", 0},
		{"WarpSpeed", formatDogmaValueParams{value: 1.2345, unitID: app.EveUnitWarpSpeed}, "1.23 AU/s", 0},
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
			assert.EqualValues(t, tc.s, s)
			assert.EqualValues(t, tc.v, v)
		})
	}
}
