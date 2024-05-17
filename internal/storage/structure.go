package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateStructureParams struct {
	ID               int64
	EveSolarSystemID int32
	EveTypeID        sql.NullInt32
	Name             string
	OwnerID          int32
	Position         model.Position
}

func (r *Storage) CreateStructure(ctx context.Context, arg CreateStructureParams) error {
	if arg.ID == 0 {
		return fmt.Errorf("invalid structure ID %d", arg.ID)
	}
	arg2 := queries.CreateStructureParams{
		ID:               int64(arg.ID),
		EveSolarSystemID: int64(arg.EveSolarSystemID),
		EveTypeID:        sql.NullInt64{Int64: int64(arg.EveTypeID.Int32), Valid: arg.EveTypeID.Valid},
		Name:             arg.Name,
		OwnerID:          int64(arg.OwnerID),
		PositionX:        arg.Position.X,
		PositionY:        arg.Position.Y,
		PositionZ:        arg.Position.Z,
	}
	if err := r.q.CreateStructure(ctx, arg2); err != nil {
		return fmt.Errorf("failed to create Structure %v, %w", arg2, err)
	}
	return nil
}

func (r *Storage) GetStructure(ctx context.Context, id int64) (*model.Structure, error) {
	row, err := r.q.GetStructure(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get Structure for id %d: %w", id, err)
	}
	s, err := r.structureFromDBModel(
		ctx,
		row.Structure,
		row.EveEntity,
		row.EveSolarSystem,
		row.EveConstellation,
		row.EveRegion,
		row.EveTypeID,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *Storage) structureFromDBModel(
	ctx context.Context,
	structure queries.Structure,
	owner queries.EveEntity,
	system queries.EveSolarSystem,
	constellation queries.EveConstellation,
	region queries.EveRegion,
	eveTypeID sql.NullInt64,
) (*model.Structure, error) {
	s := &model.Structure{
		ID:          structure.ID,
		Name:        structure.Name,
		Owner:       eveEntityFromDBModel(owner),
		Position:    model.Position{X: structure.PositionX, Y: structure.PositionY, Z: structure.PositionZ},
		SolarSystem: eveSolarSystemFromDBModel(system, constellation, region),
	}
	if eveTypeID.Valid {
		x, err := r.GetEveType(ctx, int32(eveTypeID.Int64))
		if err != nil {
			return nil, err
		}
		s.Type = x
	}
	return s, nil
}
