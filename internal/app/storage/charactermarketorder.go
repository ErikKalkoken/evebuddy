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
	"github.com/ErikKalkoken/evebuddy/internal/xmaps"
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
		States:        []string{orderStatusToDBValue[app.OrderOpen]},
	})
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
		States:      []string{orderStatusToDBValue[app.OrderOpen]},
		CharacterID: characterID,
	})
	if err != nil {
		return 0, wrapErr(err)
	}
	return v.Float64, nil
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

// CharacterMarketOrderParams describes a market order of a character.
type CharacterMarketOrderParams struct {
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

func (x CharacterMarketOrderParams) isValid() bool {
	return x.Duration != 0 &&
		!x.Issued.IsZero() &&
		x.LocationID != 0 &&
		x.OrderID != 0 &&
		x.OwnerID != 0 &&
		x.RegionID != 0 &&
		x.TypeID != 0
}

// characterMarketOrderState holds the columns of a market order which can change after it was created.
type characterMarketOrderState struct {
	escrow        optional.Optional[float64]
	price         float64
	state         app.MarketOrderState
	volumeRemains int64
}

// equal reports whether s and other are equal.
func (s characterMarketOrderState) equal(other characterMarketOrderState) bool {
	return optional.Equal(s.escrow, other.escrow) &&
		s.price == other.price &&
		s.state == other.state &&
		s.volumeRemains == other.volumeRemains
}

// ReplaceCharacterMarketOrders replaces the market orders of a character in one transaction.
// It writes only new or changed orders and marks open orders missing in args as unknown.
// Closed orders issued longer ago than retention are deleted. A retention of 0 keeps all orders.
// It reports whether anything has changed.
// Order IDs in args must be unique.
func (st *Storage) ReplaceCharacterMarketOrders(ctx context.Context, characterID int64, retention time.Duration, args []CharacterMarketOrderParams) (bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceCharacterMarketOrders: character %d: %w", characterID, err)
	}
	if characterID == 0 || retention < 0 {
		return false, wrapErr(app.ErrInvalid)
	}
	for _, arg := range args {
		if !arg.isValid() {
			return false, wrapErr(fmt.Errorf("order %d: %w", arg.OrderID, app.ErrInvalid))
		}
	}
	rows, err := st.qRO.ListCharacterMarketOrderStates(ctx, characterID)
	if err != nil {
		return false, wrapErr(err)
	}
	type currentOrder struct {
		characterMarketOrderState
		issued time.Time
	}
	current := make(map[int64]currentOrder, len(rows))
	for _, r := range rows {
		current[r.OrderID] = currentOrder{
			characterMarketOrderState: characterMarketOrderState{
				escrow:        optional.FromNullFloat64(r.Escrow),
				price:         r.Price,
				state:         orderStatusFromDBValue[r.State],
				volumeRemains: r.VolumeRemains,
			},
			issued: r.Issued,
		}
	}
	now := time.Now()
	isStale := func(state app.MarketOrderState, issued time.Time) bool {
		return retention > 0 && state != app.OrderOpen && now.Sub(issued) > retention
	}
	var changed []CharacterMarketOrderParams
	var stale, orphaned []int64
	for _, arg := range args {
		o, found := xmaps.Pop(current, arg.OrderID) // what remains in current is missing in args
		issued := arg.Issued
		if found {
			issued = o.issued // issued is not updated for existing orders
		}
		if isStale(arg.State, issued) {
			if found {
				stale = append(stale, arg.OrderID)
			}
			continue
		}
		if found && o.equal(characterMarketOrderState{
			escrow:        arg.Escrow,
			price:         arg.Price,
			state:         arg.State,
			volumeRemains: arg.VolumeRemains,
		}) {
			continue
		}
		changed = append(changed, arg)
	}
	for id, o := range current {
		if o.state == app.OrderOpen {
			if isStale(app.OrderUnknown, o.issued) {
				stale = append(stale, id)
			} else {
				// Orders can vanish from the open orders before they appear in the cached history.
				// They should get their final state from the history later.
				orphaned = append(orphaned, id)
			}
		} else if isStale(o.state, o.issued) {
			stale = append(stale, id)
		}
	}
	if len(changed) == 0 && len(stale) == 0 && len(orphaned) == 0 {
		return false, nil
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return false, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	for ids := range slices.Chunk(stale, st.MaxIDsPerQuery) {
		err := qtx.DeleteCharacterMarketOrders(ctx, queries.DeleteCharacterMarketOrdersParams{
			CharacterID: characterID,
			OrderIds:    ids,
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("delete %d orders: %w", len(ids), err))
		}
	}
	for ids := range slices.Chunk(orphaned, st.MaxIDsPerQuery) {
		err := qtx.UpdateCharacterMarketOrderState(ctx, queries.UpdateCharacterMarketOrderStateParams{
			CharacterID: characterID,
			OrderIds:    ids,
			State:       orderStatusToDBValue[app.OrderUnknown],
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("mark %d orders as unknown: %w", len(ids), err))
		}
	}
	for _, arg := range changed {
		if err := updateOrCreateCharacterMarketOrder(ctx, qtx, characterID, arg); err != nil {
			return false, fmt.Errorf("ReplaceCharacterMarketOrders: %w", err) // err already names the character
		}
	}
	if err := tx.Commit(); err != nil {
		return false, wrapErr(err)
	}
	slog.Info("Character market orders replaced", "characterID", characterID, "changed", len(changed), "orphaned", len(orphaned), "deleted", len(stale))
	return true, nil
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
	p := CharacterMarketOrderParams{
		Duration:      arg.Duration,
		Escrow:        arg.Escrow,
		IsBuyOrder:    arg.IsBuyOrder,
		IsCorporation: arg.IsCorporation,
		Issued:        arg.Issued,
		LocationID:    arg.LocationID,
		MinVolume:     arg.MinVolume,
		OrderID:       arg.OrderID,
		OwnerID:       arg.OwnerID,
		Price:         arg.Price,
		Range:         arg.Range,
		RegionID:      arg.RegionID,
		State:         arg.State,
		TypeID:        arg.TypeID,
		VolumeRemains: arg.VolumeRemains,
		VolumeTotal:   arg.VolumeTotal,
	}
	if arg.CharacterID == 0 || !p.isValid() {
		return fmt.Errorf("UpdateOrCreateCharacterMarketOrder: %+v: %w", struct {
			CharacterID int64
			OrderID     int64
		}{arg.CharacterID, arg.OrderID}, app.ErrInvalid)
	}
	return updateOrCreateCharacterMarketOrder(ctx, st.qRW, arg.CharacterID, p)
}

// updateOrCreateCharacterMarketOrder creates or updates a market order.
// It expects valid params.
func updateOrCreateCharacterMarketOrder(ctx context.Context, q *queries.Queries, characterID int64, arg CharacterMarketOrderParams) error {
	err := q.UpdateOrCreateCharacterMarketOrder(ctx, queries.UpdateOrCreateCharacterMarketOrderParams{
		CharacterID:   characterID,
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
		return fmt.Errorf("UpdateOrCreateCharacterMarketOrder: %+v: %w", struct {
			CharacterID int64
			OrderID     int64
		}{characterID, arg.OrderID}, err)
	}
	return nil
}
