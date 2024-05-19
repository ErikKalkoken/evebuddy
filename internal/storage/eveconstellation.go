package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateEveConstellationParams struct {
	ID       int32
	Name     string
	RegionID int32
}

func (r *Storage) CreateEveConstellation(ctx context.Context, arg CreateEveConstellationParams) error {
	if arg.ID == 0 {
		return fmt.Errorf("invalid ID %d", arg.ID)
	}
	arg2 := queries.CreateEveConstellationParams{
		ID:          int64(arg.ID),
		EveRegionID: int64(arg.RegionID),
		Name:        arg.Name,
	}
	err := r.q.CreateEveConstellation(ctx, arg2)
	if err != nil {
		return fmt.Errorf("failed to create EveConstellation %v, %w", arg, err)
	}
	return nil
}

func (r *Storage) GetEveConstellation(ctx context.Context, id int32) (*model.EveConstellation, error) {
	row, err := r.q.GetEveConstellation(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get EveConstellation for id %d: %w", id, err)
	}
	g := eveConstellationFromDBModel(row.EveConstellation, row.EveRegion)
	return g, nil
}

func eveConstellationFromDBModel(c queries.EveConstellation, r queries.EveRegion) *model.EveConstellation {
	return &model.EveConstellation{
		ID:     int32(c.ID),
		Name:   c.Name,
		Region: eveRegionFromDBModel(r),
	}
}
