package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	islices "github.com/ErikKalkoken/evebuddy/internal/helper/slices"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateEveTypeParams struct {
	ID             int32
	GroupID        int32
	Capacity       float32
	Description    string
	GraphicID      int32
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

func (r *Storage) CreateEveType(ctx context.Context, arg CreateEveTypeParams) error {
	if arg.ID == 0 {
		return fmt.Errorf("invalid ID %d", arg.ID)
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
	err := r.q.CreateEveType(ctx, arg2)
	if err != nil {
		return fmt.Errorf("failed to create EveType %v, %w", arg, err)
	}
	return nil
}

func (r *Storage) GetEveType(ctx context.Context, id int32) (*model.EveType, error) {
	row, err := r.q.GetEveType(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get EveType for id %d: %w", id, err)
	}
	t := eveTypeFromDBModel(row.EveType, row.EveGroup, row.EveCategory)
	return t, nil
}

func (r *Storage) MissingEveTypes(ctx context.Context, ids []int32) ([]int32, error) {
	currentIDs, err := r.q.ListEveTypeIDs(ctx)
	if err != nil {
		return nil, err
	}
	current := set.NewFromSlice(islices.ConvertNumeric[int64, int32](currentIDs))
	incoming := set.NewFromSlice(ids)
	missing := incoming.Difference(current)
	return missing.ToSlice(), nil
}

func eveTypeFromDBModel(t queries.EveType, g queries.EveGroup, c queries.EveCategory) *model.EveType {
	return &model.EveType{
		ID:             int32(t.ID),
		Group:          eveGroupFromDBModel(g, c),
		Capacity:       float32(t.Capacity),
		Description:    t.Description,
		GraphicID:      int32(t.GraphicID),
		IconID:         int32(t.GraphicID),
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
