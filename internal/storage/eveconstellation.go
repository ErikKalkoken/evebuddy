package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

func (r *Storage) CreateEveConstellation(ctx context.Context, id, eve_region_id int32, name string) error {
	if id == 0 {
		return fmt.Errorf("invalid ID %d", id)
	}
	arg := queries.CreateEveConstellationParams{
		ID:          int64(id),
		EveRegionID: int64(eve_region_id),
		Name:        name,
	}
	err := r.q.CreateEveConstellation(ctx, arg)
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
