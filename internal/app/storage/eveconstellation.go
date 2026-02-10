package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveConstellationParams struct {
	ID       int64
	Name     string
	RegionID int64
}

func (st *Storage) CreateEveConstellation(ctx context.Context, arg CreateEveConstellationParams) error {
	if arg.ID == 0 || arg.RegionID == 0 {
		return fmt.Errorf("CreateEveConstellation: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateEveConstellationParams{
		ID:         arg.ID,
		EveRegionID:arg.RegionID,
		Name:        arg.Name,
	}
	err := st.qRW.CreateEveConstellation(ctx, arg2)
	if err != nil {
		return fmt.Errorf("CreateEveConstellation: %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetEveConstellation(ctx context.Context, id int64) (*app.EveConstellation, error) {
	row, err := st.qRO.GetEveConstellation(ctx,id)
	if err != nil {
		return nil, fmt.Errorf("get EveConstellation for id %d: %w", id, convertGetError(err))
	}
	g := eveConstellationFromDBModel(row.EveConstellation, row.EveRegion)
	return g, nil
}

func eveConstellationFromDBModel(c queries.EveConstellation, r queries.EveRegion) *app.EveConstellation {
	return &app.EveConstellation{
		ID:    c.ID,
		Name:   c.Name,
		Region: eveRegionFromDBModel(r),
	}
}
