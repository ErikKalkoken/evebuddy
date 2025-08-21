package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

var orderStatusFromDBValue = map[string]app.MarketOrderState{
	"":          app.OrderUndefined,
	"cancelled": app.OrderCancelled,
	"expired":   app.OrderExpired,
	"open":      app.OrderOpen,
}

var orderStatusToDBValue = map[app.MarketOrderState]string{}

func init() {
	for k, v := range orderStatusFromDBValue {
		orderStatusToDBValue[v] = k
	}
}

func (st *Storage) GetCharacterMarketOrder(ctx context.Context, characterID int32, orderID int64) (*app.CharacterMarketOrder, error) {
	arg := queries.GetCharacterMarketOrderParams{
		CharacterID: int64(characterID),
		OrderID:     orderID,
	}
	r, err := st.qRO.GetCharacterMarketOrder(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("GetCharacterMarketOrder for character %d: %w", characterID, convertGetError(err))
	}
	o := characterMarketOrderFromDBModel(characterMarketOrderFromDBModelParams{
		cmo:          r.CharacterMarketOrder,
		locationName: r.LocationName,
		regionName:   r.RegionName,
		typeName:     r.TypeName,
	})
	return o, err
}

func (st *Storage) ListAllCharacterMarketOrders(ctx context.Context, isBuyOrders bool) ([]*app.CharacterMarketOrder, error) {
	rows, err := st.qRO.ListAllCharacterMarketOrders(ctx, isBuyOrders)
	if err != nil {
		return nil, fmt.Errorf("ListAllCharacterMarketOrders: %w", err)
	}
	oo := make([]*app.CharacterMarketOrder, len(rows))
	for i, r := range rows {
		oo[i] = characterMarketOrderFromDBModel(characterMarketOrderFromDBModelParams{
			cmo:          r.CharacterMarketOrder,
			locationName: r.LocationName,
			regionName:   r.RegionName,
			typeName:     r.TypeName,
		})
	}
	return oo, nil
}

func (st *Storage) ListCharacterMarketOrders(ctx context.Context, characterID int32) ([]*app.CharacterMarketOrder, error) {
	rows, err := st.qRO.ListCharacterMarketOrders(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("ListCharacterMarketOrder for character %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterMarketOrder, len(rows))
	for i, r := range rows {
		oo[i] = characterMarketOrderFromDBModel(characterMarketOrderFromDBModelParams{
			cmo:          r.CharacterMarketOrder,
			locationName: r.LocationName,
			regionName:   r.RegionName,
			typeName:     r.TypeName,
		})
	}
	return oo, nil
}

type characterMarketOrderFromDBModelParams struct {
	cmo          queries.CharacterMarketOrder
	locationName string
	regionName   string
	typeName     string
}

func characterMarketOrderFromDBModel(arg characterMarketOrderFromDBModelParams) *app.CharacterMarketOrder {
	o2 := &app.CharacterMarketOrder{
		CharacterID:   int32(arg.cmo.CharacterID),
		Duration:      int(arg.cmo.Duration),
		Escrow:        optional.FromNullFloat64(arg.cmo.Escrow),
		IsBuyOrder:    arg.cmo.IsBuyOrder,
		IsCorporation: arg.cmo.IsCorporation,
		Issued:        arg.cmo.Issued,
		Location: &app.EntityShort[int64]{
			ID:   arg.cmo.LocationID,
			Name: arg.locationName,
		},
		MinVolume: optional.FromNullInt64ToInteger[int](arg.cmo.MinVolume),
		OrderID:   arg.cmo.OrderID,
		Price:     arg.cmo.Price,
		Range:     arg.cmo.Range,
		Region: &app.EntityShort[int32]{
			ID:   int32(arg.cmo.RegionID),
			Name: arg.regionName,
		},
		State: orderStatusFromDBValue[arg.cmo.State],
		Type: &app.EntityShort[int32]{
			ID:   int32(arg.cmo.TypeID),
			Name: arg.typeName,
		},
		VolumeRemains: int(arg.cmo.VolumeRemains),
		VolumeTotal:   int(arg.cmo.VolumeTotal),
	}
	return o2
}

type UpdateOrCreateCharacterMarketOrderParams struct {
	CharacterID   int32
	Duration      int
	Escrow        optional.Optional[float64]
	IsBuyOrder    bool
	IsCorporation bool
	Issued        time.Time
	LocationID    int64
	MinVolume     optional.Optional[int]
	OrderID       int64
	Price         float64
	Range         string
	RegionID      int32
	State         app.MarketOrderState
	TypeID        int32
	VolumeRemains int
	VolumeTotal   int
}

func (st *Storage) UpdateOrCreateCharacterMarketOrder(ctx context.Context, arg UpdateOrCreateCharacterMarketOrderParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCharacterMarketOrder: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 ||
		arg.Duration == 0 ||
		arg.Issued.IsZero() ||
		arg.LocationID == 0 ||
		arg.OrderID == 0 ||
		arg.RegionID == 0 ||
		arg.TypeID == 0 ||
		arg.VolumeTotal == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateCharacterMarketOrder(ctx, queries.UpdateOrCreateCharacterMarketOrderParams{
		CharacterID:   int64(arg.CharacterID),
		Duration:      int64(arg.Duration),
		Escrow:        optional.ToNullFloat64(arg.Escrow),
		IsBuyOrder:    arg.IsBuyOrder,
		IsCorporation: arg.IsCorporation,
		Issued:        arg.Issued,
		LocationID:    arg.LocationID,
		MinVolume:     optional.ToNullInt64(arg.MinVolume),
		OrderID:       arg.OrderID,
		Price:         arg.Price,
		Range:         arg.Range,
		RegionID:      int64(arg.RegionID),
		State:         orderStatusToDBValue[arg.State],
		TypeID:        int64(arg.TypeID),
		VolumeRemains: int64(arg.VolumeRemains),
		VolumeTotal:   int64(arg.VolumeTotal),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}
