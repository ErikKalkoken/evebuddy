package storage

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateEveTypeParams struct {
	ID             int64
	Capacity       optional.Optional[float64]
	Description    string
	GraphicID      optional.Optional[int64]
	GroupID        int64
	IconID         optional.Optional[int64]
	IsPublished    bool
	MarketGroupID  optional.Optional[int64]
	Mass           optional.Optional[float64]
	Name           string
	PackagedVolume optional.Optional[float64]
	PortionSize    optional.Optional[int64]
	Radius         optional.Optional[float64]
	Volume         optional.Optional[float64]
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
		ID:             arg.ID,
		EveGroupID:     arg.GroupID,
		Capacity:       arg.Capacity.ValueOrZero(),
		Description:    arg.Description,
		GraphicID:      arg.GraphicID.ValueOrZero(),
		IconID:         arg.IconID.ValueOrZero(),
		IsPublished:    arg.IsPublished,
		MarketGroupID:  arg.MarketGroupID.ValueOrZero(),
		Mass:           arg.Mass.ValueOrZero(),
		Name:           arg.Name,
		PackagedVolume: arg.PackagedVolume.ValueOrZero(),
		PortionSize:    arg.PortionSize.ValueOrZero(),
		Radius:         arg.Radius.ValueOrZero(),
		Volume:         arg.Volume.ValueOrZero(),
	}
	err := q.CreateEveType(ctx, arg2)
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetEveType(ctx context.Context, id int64) (*app.EveType, error) {
	return getEveType(ctx, st.qRO, id)
}

func getEveType(ctx context.Context, q *queries.Queries, id int64) (*app.EveType, error) {
	r, err := q.GetEveType(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getEveType for id %d: %w", id, convertGetError(err))
	}
	return eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory), nil
}

func (st *Storage) ListEveTypes(ctx context.Context) ([]*app.EveType, error) {
	rows, err := st.qRO.ListEveTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListEveTypes: %w", err)
	}
	var oo []*app.EveType
	for _, r := range rows {
		oo = append(oo, eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory))
	}
	return oo, nil
}

func (st *Storage) ListEveSkills(ctx context.Context) ([]*app.EveType, error) {
	rows, err := st.qRO.ListEveSkills(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListEveSkills: %w", err)
	}
	var oo []*app.EveType
	for _, r := range rows {
		oo = append(oo, eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory))
	}
	return oo, nil
}

func eveTypeFromDBModel(t queries.EveType, g queries.EveGroup, c queries.EveCategory) *app.EveType {
	return &app.EveType{
		ID:             t.ID,
		Group:          eveGroupFromDBModel(g, c),
		Capacity:       optional.FromZeroValue(t.Capacity),
		Description:    t.Description,
		GraphicID:      optional.FromZeroValue(t.GraphicID),
		IconID:         optional.FromZeroValue(t.IconID),
		IsPublished:    t.IsPublished,
		MarketGroupID:  optional.FromZeroValue(t.MarketGroupID),
		Mass:           optional.FromZeroValue(t.Mass),
		Name:           t.Name,
		PackagedVolume: optional.FromZeroValue(t.PackagedVolume),
		PortionSize:    optional.FromZeroValue(t.PortionSize),
		Radius:         optional.FromZeroValue(t.Radius),
		Volume:         optional.FromZeroValue(t.Volume),
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

func (st *Storage) ListEveTypeIDs(ctx context.Context) (set.Set[int64], error) {
	ids, err := st.qRO.ListEveTypeIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("ListEveTypeIDs: %w", err)
	}
	ids2 := set.Collect(slices.Values(ids))
	return ids2, nil
}

func (st *Storage) MissingEveTypes(ctx context.Context, ids set.Set[int64]) (set.Set[int64], error) {
	currentIDs, err := st.qRO.ListEveTypeIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, err
	}
	current := set.Collect(slices.Values(currentIDs))
	missing := set.Difference(ids, current)
	return missing, nil
}
