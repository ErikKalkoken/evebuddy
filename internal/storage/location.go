package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type UpdateOrCreateLocationParams struct {
	ID               int64
	EveSolarSystemID sql.NullInt32
	EveTypeID        sql.NullInt32
	Name             string
	OwnerID          sql.NullInt32
	UpdatedAt        time.Time
}

func (r *Storage) UpdateOrCreateLocation(ctx context.Context, arg UpdateOrCreateLocationParams) error {
	if arg.ID == 0 {
		return fmt.Errorf("invalid structure ID %d", arg.ID)
	}
	arg2 := queries.UpdateOrCreateLocationParams{
		ID:               int64(arg.ID),
		EveSolarSystemID: sql.NullInt64{Int64: int64(arg.EveSolarSystemID.Int32), Valid: arg.EveSolarSystemID.Valid},
		EveTypeID:        sql.NullInt64{Int64: int64(arg.EveTypeID.Int32), Valid: arg.EveTypeID.Valid},
		Name:             arg.Name,
		OwnerID:          sql.NullInt64{Int64: int64(arg.OwnerID.Int32), Valid: arg.OwnerID.Valid},
		UpdatedAt:        arg.UpdatedAt,
	}
	if err := r.q.UpdateOrCreateLocation(ctx, arg2); err != nil {
		return fmt.Errorf("failed to update or create Structure %v, %w", arg, err)
	}
	return nil
}

func (r *Storage) GetLocation(ctx context.Context, id int64) (*model.Location, error) {
	l, err := r.q.GetLocation(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get Structure for id %d: %w", id, err)
	}
	x, err := r.locationFromDBModel(ctx, l)
	if err != nil {
		return nil, err
	}
	return x, nil
}

func (r *Storage) MissingLocations(ctx context.Context, ids []int64) ([]int64, error) {
	currentIDs, err := r.q.ListLocationIDs(ctx)
	if err != nil {
		return nil, err
	}
	current := set.NewFromSlice(currentIDs)
	incoming := set.NewFromSlice(ids)
	missing := incoming.Difference(current)
	return missing.ToSlice(), nil
}

func (r *Storage) locationFromDBModel(ctx context.Context, l queries.Location) (*model.Location, error) {
	l2 := &model.Location{
		ID:        l.ID,
		Name:      l.Name,
		UpdatedAt: l.UpdatedAt,
	}
	if l.EveTypeID.Valid {
		x, err := r.GetEveType(ctx, int32(l.EveTypeID.Int64))
		if err != nil {
			return nil, err
		}
		l2.Type = x
	}
	if l.EveSolarSystemID.Valid {
		x, err := r.GetEveSolarSystem(ctx, int32(l.EveSolarSystemID.Int64))
		if err != nil {
			return nil, err
		}
		l2.SolarSystem = x
	}
	if l.OwnerID.Valid {
		x, err := r.GetEveEntity(ctx, int32(l.OwnerID.Int64))
		if err != nil {
			return nil, err
		}
		l2.Owner = x
	}
	return l2, nil
}
