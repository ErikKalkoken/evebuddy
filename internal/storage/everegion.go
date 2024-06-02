package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateEveRegionParams struct {
	Description string
	ID          int32
	Name        string
}

func (st *Storage) CreateEveRegion(ctx context.Context, arg CreateEveRegionParams) (*model.EveRegion, error) {
	if arg.ID == 0 {
		return nil, fmt.Errorf("invalid ID %d", arg.ID)
	}
	arg2 := queries.CreateEveRegionParams{
		ID:          int64(arg.ID),
		Description: arg.Description,
		Name:        arg.Name,
	}
	e, err := st.q.CreateEveRegion(ctx, arg2)
	if err != nil {
		return nil, fmt.Errorf("failed to create EveRegion %v, %w", arg2, err)
	}
	return eveRegionFromDBModel(e), nil
}

func (st *Storage) GetEveRegion(ctx context.Context, id int32) (*model.EveRegion, error) {
	c, err := st.q.GetEveRegion(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get EveRegion for id %d: %w", id, err)
	}
	return eveRegionFromDBModel(c), nil
}

func eveRegionFromDBModel(c queries.EveRegion) *model.EveRegion {
	return &model.EveRegion{
		ID:          int32(c.ID),
		Description: c.Description,
		Name:        c.Name,
	}
}
