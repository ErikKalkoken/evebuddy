package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

var orderStatusFromDBValue = map[string]app.MarketOrderState{
	"":          app.OrderUndefined,
	"cancelled": app.OrderCancelled,
	"expired":   app.OrderExpired,
	"open":      app.OrderOpen,
	"unknown":   app.OrderUnknown,
}

var orderStatusToDBValue = map[app.MarketOrderState]string{}

func init() {
	for k, v := range orderStatusFromDBValue {
		orderStatusToDBValue[v] = k
	}
}

func (st *Storage) DeleteCharacterMarketOrders(ctx context.Context, characterID int64, orderIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCharacterMarketOrdersByID for character %d and job IDs: %v: %w", characterID, orderIDs, err)
	}
	if characterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if orderIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.DeleteCharacterMarketOrders(ctx, queries.DeleteCharacterMarketOrdersParams{
		CharacterID: characterID,
		OrderIds:    slices.Collect(orderIDs.All()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Market jobs deleted", "characterID", characterID, "jobIDs", orderIDs)
	return nil
}

func (st *Storage) GetCharacterMarketOrder(ctx context.Context, characterID int64, orderID int64) (*app.CharacterMarketOrder, error) {
	r, err := st.qRO.GetCharacterMarketOrder(ctx, queries.GetCharacterMarketOrderParams{
		CharacterID: characterID,
		OrderID:     orderID,
	})
	if err != nil {
		return nil, fmt.Errorf("GetCharacterMarketOrder for character %d: %w", characterID, convertGetError(err))
	}
	o := characterMarketOrderFromDBModel(characterMarketOrderFromDBModelParams{
		cmo:              r.CharacterMarketOrder,
		locationName:     r.LocationName,
		locationSecurity: r.LocationSecurity,
		owner:            r.EveEntity,
		regionName:       r.RegionName,
		typeName:         r.TypeName,
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
			cmo:              r.CharacterMarketOrder,
			locationName:     r.LocationName,
			locationSecurity: r.LocationSecurity,
			owner:            r.EveEntity,
			regionName:       r.RegionName,
			typeName:         r.TypeName,
		})
	}
	return oo, nil
}

func (st *Storage) ListCharacterMarketOrderIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterMarketOrderIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("ListCharacterMarketOrderIDs for character %d: %w", characterID, err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListCharacterMarketOrders(ctx context.Context, characterID int64) ([]*app.CharacterMarketOrder, error) {
	rows, err := st.qRO.ListCharacterMarketOrders(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("ListCharacterMarketOrder for character %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterMarketOrder, len(rows))
	for i, r := range rows {
		oo[i] = characterMarketOrderFromDBModel(characterMarketOrderFromDBModelParams{
			cmo:              r.CharacterMarketOrder,
			locationName:     r.LocationName,
			locationSecurity: r.LocationSecurity,
			owner:            r.EveEntity,
			regionName:       r.RegionName,
			typeName:         r.TypeName,
		})
	}
	return oo, nil
}

type characterMarketOrderFromDBModelParams struct {
	cmo              queries.CharacterMarketOrder
	locationName     string
	locationSecurity sql.NullFloat64
	owner            queries.EveEntity
	regionName       string
	typeName         string
}

func characterMarketOrderFromDBModel(arg characterMarketOrderFromDBModelParams) *app.CharacterMarketOrder {
	o2 := &app.CharacterMarketOrder{
		CharacterID:   arg.cmo.CharacterID,
		Duration:      arg.cmo.Duration,
		Escrow:        optional.FromNullFloat64(arg.cmo.Escrow),
		IsBuyOrder:    optional.FromZeroValue(arg.cmo.IsBuyOrder),
		IsCorporation: arg.cmo.IsCorporation,
		Issued:        arg.cmo.Issued,
		Location: &app.EveLocationShort{
			ID:             arg.cmo.LocationID,
			Name:           optional.New(arg.locationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.locationSecurity),
		},
		MinVolume: optional.FromNullInt64(arg.cmo.MinVolume),
		OrderID:   arg.cmo.OrderID,
		Owner:     eveEntityFromDBModel(arg.owner),
		Price:     arg.cmo.Price,
		Range:     arg.cmo.Range,
		Region: &app.EntityShort[int64]{
			ID:   arg.cmo.RegionID,
			Name: arg.regionName,
		},
		State: orderStatusFromDBValue[arg.cmo.State],
		Type: &app.EntityShort[int64]{
			ID:   arg.cmo.TypeID,
			Name: arg.typeName,
		},
		VolumeRemains: arg.cmo.VolumeRemains,
		VolumeTotal:   arg.cmo.VolumeTotal,
	}
	return o2
}

type UpdateCharacterMarketOrderStateParams struct {
	CharacterID int64
	OrderIDs    set.Set[int64]
	State       app.MarketOrderState
}

func (st *Storage) UpdateCharacterMarketOrderState(ctx context.Context, arg UpdateCharacterMarketOrderStateParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateCharacterMarketOrderState %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.OrderIDs.Contains(0) {
		return wrapErr(app.ErrInvalid)
	}
	if arg.OrderIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.UpdateCharacterMarketOrderState(ctx, queries.UpdateCharacterMarketOrderStateParams{
		CharacterID: arg.CharacterID,
		OrderIds:    slices.Collect(arg.OrderIDs.All()),
		State:       orderStatusToDBValue[arg.State],
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

type UpdateOrCreateCharacterMarketOrderParams struct {
	CharacterID   int64
	Duration      int64
	Escrow        optional.Optional[float64]
	IsBuyOrder    optional.Optional[bool]
	IsCorporation bool
	Issued        time.Time
	LocationID    int64
	MinVolume     optional.Optional[int64]
	OrderID       int64
	OwnerID       int64
	Price         float64
	Range         string
	RegionID      int64
	State         app.MarketOrderState
	TypeID        int64
	VolumeRemains int64
	VolumeTotal   int64
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
		arg.OwnerID == 0 ||
		arg.RegionID == 0 ||
		arg.TypeID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateCharacterMarketOrder(ctx, queries.UpdateOrCreateCharacterMarketOrderParams{
		CharacterID:   arg.CharacterID,
		Duration:      arg.Duration,
		Escrow:        optional.ToNullFloat64(arg.Escrow),
		IsBuyOrder:    arg.IsBuyOrder.ValueOrZero(),
		IsCorporation: arg.IsCorporation,
		Issued:        arg.Issued,
		LocationID:    arg.LocationID,
		MinVolume:     optional.ToNullInt64(arg.MinVolume),
		OrderID:       arg.OrderID,
		OwnerID:       arg.OwnerID,
		Price:         arg.Price,
		Range:         arg.Range,
		RegionID:      arg.RegionID,
		State:         orderStatusToDBValue[arg.State],
		TypeID:        arg.TypeID,
		VolumeRemains: arg.VolumeRemains,
		VolumeTotal:   arg.VolumeTotal,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}
