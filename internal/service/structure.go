package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

// getOrCreateStructureESI return a structure when it already exists
// or else tries to fetch and create a new structure from ESI.
//
// Important: A token with the structure scope must be set in the context
func (s *Service) getOrCreateStructureESI(ctx context.Context, id int64) (*model.Structure, error) {
	x, err := s.r.GetStructure(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.createStructureFromESI(ctx, id)
		}
		return x, err
	}
	return x, nil
}

// createStructureFromESI tries to fetch and create a new structure from ESI.
//
// Important: A token with the structure scope must be set in the context
func (s *Service) createStructureFromESI(ctx context.Context, id int64) (*model.Structure, error) {
	key := fmt.Sprintf("createStructureFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		structure, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStructuresStructureId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		system, err := s.getOrCreateEveSolarSystemESI(ctx, structure.SolarSystemId)
		if err != nil {
			return nil, err
		}
		_, err = s.AddMissingEveEntities(ctx, []int32{structure.OwnerId})
		if err != nil {
			return nil, err
		}
		arg := storage.CreateStructureParams{
			ID:               id,
			EveSolarSystemID: system.ID,
			Name:             structure.Name,
			OwnerID:          structure.OwnerId,
			Position: model.Position{
				X: structure.Position.X,
				Y: structure.Position.Y,
				Z: structure.Position.Z,
			},
		}
		if structure.TypeId != 0 {
			myType, err := s.getOrCreateEveTypeESI(ctx, structure.TypeId)
			if err != nil {
				return nil, err
			}
			arg.EveTypeID = sql.NullInt32{Int32: myType.ID, Valid: true}
		}
		if err := s.r.CreateStructure(ctx, arg); err != nil {
			return nil, err
		}
		return s.r.GetStructure(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.Structure), nil
}
