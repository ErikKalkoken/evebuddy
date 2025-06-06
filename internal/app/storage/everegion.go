package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveRegionParams struct {
	Description string
	ID          int32
	Name        string
}

func (st *Storage) CreateEveRegion(ctx context.Context, arg CreateEveRegionParams) (*app.EveRegion, error) {
	if arg.ID == 0 {
		return nil, fmt.Errorf("CreateEveRegion for id %d: %w", arg.ID, app.ErrInvalid)
	}
	arg2 := queries.CreateEveRegionParams{
		ID:          int64(arg.ID),
		Description: arg.Description,
		Name:        arg.Name,
	}
	e, err := st.qRW.CreateEveRegion(ctx, arg2)
	if err != nil {
		return nil, fmt.Errorf("CreateEveRegion for id %d: %w", arg.ID, err)
	}
	return eveRegionFromDBModel(e), nil
}

func (st *Storage) GetEveRegion(ctx context.Context, id int32) (*app.EveRegion, error) {
	c, err := st.qRO.GetEveRegion(ctx, int64(id))
	if err != nil {
		return nil, fmt.Errorf("get EveRegion for id %d: %w", id, convertGetError(err))
	}
	return eveRegionFromDBModel(c), nil
}

func eveRegionFromDBModel(c queries.EveRegion) *app.EveRegion {
	return &app.EveRegion{
		ID:          int32(c.ID),
		Description: c.Description,
		Name:        c.Name,
	}
}
