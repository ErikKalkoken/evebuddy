package storage

import (
	"context"
	"database/sql"
	"errors"
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

func (st *Storage) CalculateCharacterOrderItemsValue(ctx context.Context, characterID int64) (float64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CalculateCharacterOrderItemsValue: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	v, err := st.qRO.CalculateCharacterOrderItemsValue(ctx, queries.CalculateCharacterOrderItemsValueParams{
		CharacterID:   characterID,
		EveCategoryID: app.EveCategoryBlueprint,
		States: []string{
			orderStatusToDBValue[app.OrderOpen],
			orderStatusToDBValue[app.OrderExpired],
		},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, wrapErr(err)
	}
	return v.Float64, nil
}

func (st *Storage) CalculateCharacterOrdersEscrow(ctx context.Context, characterID int64) (float64, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("CalculateCharacterOrdersEscrow: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	v, err := st.qRO.CalculateCharacterOrdersEscrow(ctx, queries.CalculateCharacterOrdersEscrowParams{
		States: []string{
			orderStatusToDBValue[app.OrderOpen],
			orderStatusToDBValue[app.OrderExpired],
		},
		CharacterID: characterID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, wrapErr(err)
	}
	return v.Float64, nil
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
	o := characterMarketOrderFromDBModel(
		r.CharacterMarketOrder,
		r.LocationName,
		r.LocationSecurity,
		r.EveEntity,
		regionName(r.RegionName),
		typeName(r.TypeName),
	)
	return o, err
}

func (st *Storage) ListAllCharacterMarketOrders(ctx context.Context, isBuyOrders bool) ([]*app.CharacterMarketOrder, error) {
	rows, err := st.qRO.ListAllCharacterMarketOrders(ctx, isBuyOrders)
	if err != nil {
		return nil, fmt.Errorf("ListAllCharacterMarketOrders: %w", err)
	}
	oo := make([]*app.CharacterMarketOrder, len(rows))
	for i, r := range rows {
		oo[i] = characterMarketOrderFromDBModel(
			r.CharacterMarketOrder,
			r.LocationName,
			r.LocationSecurity,
			r.EveEntity,
			regionName(r.RegionName),
			typeName(r.TypeName),
		)
	}
	return oo, nil
}

func (st *Storage) ListCharacterMarketOrderIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterMarketOrderIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("ListCharacterMarketOrderIDs for character %d: %w", characterID, err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) ListCharacterMarketOrders(ctx context.Context, characterID int64) ([]*app.CharacterMarketOrder, error) {
	rows, err := st.qRO.ListCharacterMarketOrders(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("ListCharacterMarketOrder for character %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterMarketOrder, len(rows))
	for i, r := range rows {
		oo[i] = characterMarketOrderFromDBModel(
			r.CharacterMarketOrder,
			r.LocationName,
			r.LocationSecurity,
			r.EveEntity,
			regionName(r.RegionName),
			typeName(r.TypeName),
		)
	}
	return oo, nil
}


func characterMarketOrderFromDBModel(
	cmo queries.CharacterMarketOrder,
	locationName string,
	locationSecurity sql.NullFloat64,
	owner queries.EveEntity,
	regionName regionName,
	typeName typeName,
) *app.CharacterMarketOrder {
	o2 := &app.CharacterMarketOrder{
		CharacterID:   cmo.CharacterID,
		Duration:      cmo.Duration,
		Escrow:        optional.FromNullFloat64(cmo.Escrow),
		IsBuyOrder:    optional.FromZeroValue(cmo.IsBuyOrder),
		IsCorporation: cmo.IsCorporation,
		Issued:        cmo.Issued,
		Location: &app.EveLocationShort{
			ID:             cmo.LocationID,
			Name:           optional.New(locationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(locationSecurity),
		},
		MinVolume: optional.FromNullInt64(cmo.MinVolume),
		OrderID:   cmo.OrderID,
		Owner:     eveEntityFromDBModel(owner),
		Price:     cmo.Price,
		Range:     cmo.Range,
		Region: &app.EntityShort{
			ID:   cmo.RegionID,
			Name: string(regionName),
		},
		State: orderStatusFromDBValue[cmo.State],
		Type: &app.EntityShort{
			ID:   cmo.TypeID,
			Name: string(typeName),
		},
		VolumeRemains: cmo.VolumeRemains,
		VolumeTotal:   cmo.VolumeTotal,
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
