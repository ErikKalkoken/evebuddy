package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type CreateEveTypeParams struct {
	ID             int32
	Capacity       float32
	Description    string
	GraphicID      int32
	GroupID        int32
	IconID         int32
	IsPublished    bool
	MarketGroupID  int32
	Mass           float32
	Name           string
	PackagedVolume float32
	PortionSize    int
	Radius         float32
	Volume         float32
}

func (st *Storage) CreateEveType(ctx context.Context, arg CreateEveTypeParams) error {
	return createEveType(ctx, st.qRW, arg)
}

func createEveType(ctx context.Context, q *queries.Queries, arg CreateEveTypeParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("createEveType: %+v: %w", arg, err)
	}

	if arg.ID == 0 || arg.GroupID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	arg2 := queries.CreateEveTypeParams{
		ID:             int64(arg.ID),
		EveGroupID:     int64(arg.GroupID),
		Capacity:       float64(arg.Capacity),
		Description:    arg.Description,
		GraphicID:      int64(arg.GraphicID),
		IconID:         int64(arg.IconID),
		IsPublished:    arg.IsPublished,
		MarketGroupID:  int64(arg.MarketGroupID),
		Mass:           float64(arg.Mass),
		Name:           arg.Name,
		PackagedVolume: float64(arg.PackagedVolume),
		PortionSize:    int64(arg.PortionSize),
		Radius:         float64(arg.Radius),
		Volume:         float64(arg.Volume),
	}
	err := q.CreateEveType(ctx, arg2)
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetEveType(ctx context.Context, id int32) (*app.EveType, error) {
	return getEveType(ctx, st.qRO, id)
}

func getEveType(ctx context.Context, q *queries.Queries, id int32) (*app.EveType, error) {
	r, err := q.GetEveType(ctx, int64(id))
	if err != nil {
		return nil, fmt.Errorf("getEveType for id %d: %w", id, convertGetError(err))
	}
	return eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory), nil
}

func eveTypeFromDBModel(t queries.EveType, g queries.EveGroup, c queries.EveCategory) *app.EveType {
	return &app.EveType{
		ID:             int32(t.ID),
		Group:          eveGroupFromDBModel(g, c),
		Capacity:       float32(t.Capacity),
		Description:    t.Description,
		GraphicID:      int32(t.GraphicID),
		IconID:         int32(t.IconID),
		IsPublished:    t.IsPublished,
		MarketGroupID:  int32(t.MarketGroupID),
		Mass:           float32(t.Mass),
		Name:           t.Name,
		PackagedVolume: float32(t.PackagedVolume),
		PortionSize:    int(t.PortionSize),
		Radius:         float32(t.Radius),
		Volume:         float32(t.Volume),
	}
}

func (st *Storage) GetOrCreateEveType(ctx context.Context, arg CreateEveTypeParams) (*app.EveType, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetOrCreateEveType: %+v: %w", arg, err)
	}
	var o *app.EveType
	tx, err := st.dbRW.Begin()
	if err != nil {
		return nil, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	o, err = getEveType(ctx, qtx, arg.ID)
	if err != nil {
		if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		err := createEveType(ctx, qtx, arg)
		if err != nil {
			return nil, err
		}
		x, err := getEveType(ctx, qtx, arg.ID)
		if err != nil {
			return nil, err
		}
		o = x
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return o, nil
}

func (st *Storage) MissingEveTypes(ctx context.Context, ids set.Set[int32]) (set.Set[int32], error) {
	currentIDs, err := st.qRO.ListEveTypeIDs(ctx)
	if err != nil {
		return set.Set[int32]{}, err
	}
	current := set.Of(convertNumericSlice[int32](currentIDs)...)
	missing := set.Difference(ids, current)
	return missing, nil
}
