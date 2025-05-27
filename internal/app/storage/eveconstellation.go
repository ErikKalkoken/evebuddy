package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateEveConstellationParams struct {
	ID       int32
	Name     string
	RegionID int32
}

func (st *Storage) CreateEveConstellation(ctx context.Context, arg CreateEveConstellationParams) error {
	if arg.ID == 0 || arg.RegionID == 0 {
		return fmt.Errorf("CreateEveConstellation: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateEveConstellationParams{
		ID:          int64(arg.ID),
		EveRegionID: int64(arg.RegionID),
		Name:        arg.Name,
	}
	err := st.qRW.CreateEveConstellation(ctx, arg2)
	if err != nil {
		return fmt.Errorf("CreateEveConstellation: %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetEveConstellation(ctx context.Context, id int32) (*app.EveConstellation, error) {
	row, err := st.qRO.GetEveConstellation(ctx, int64(id))
	if err != nil {
		return nil, fmt.Errorf("get EveConstellation for id %d: %w", id, convertGetError(err))
	}
	g := eveConstellationFromDBModel(row.EveConstellation, row.EveRegion)
	return g, nil
}

func eveConstellationFromDBModel(c queries.EveConstellation, r queries.EveRegion) *app.EveConstellation {
	return &app.EveConstellation{
		ID:     int32(c.ID),
		Name:   c.Name,
		Region: eveRegionFromDBModel(r),
	}
}
