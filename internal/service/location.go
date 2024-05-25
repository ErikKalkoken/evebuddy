package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

// getOrCreateLocationESI return a structure when it already exists
// or else tries to fetch and create a new structure from ESI.
//
// Important: A token with the structure scope must be set in the context
func (s *Service) getOrCreateLocationESI(ctx context.Context, id int64) (*model.Location, error) {
	x, err := s.r.GetLocation(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return s.updateOrCreateLocationESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

// updateOrCreateLocationESI tries to fetch and create a new structure from ESI.
//
// Important: A token with the structure scope must be set in the context when trying to fetch a structure.
func (s *Service) updateOrCreateLocationESI(ctx context.Context, id int64) (*model.Location, error) {
	key := fmt.Sprintf("createStructureFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		var arg storage.UpdateOrCreateLocationParams
		switch model.LocationVariantFromID(id) {
		case model.LocationVariantUnknown:
			t, err := s.getOrCreateEveTypeESI(ctx, model.EveTypeIDSolarSystem)
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:        id,
				EveTypeID: sql.NullInt32{Int32: t.ID, Valid: true},
			}
		case model.LocationVariantAssetSafety:
			t, err := s.getOrCreateEveTypeESI(ctx, model.EveTypeIDAssetSafetyWrap)
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:        id,
				EveTypeID: sql.NullInt32{Int32: t.ID, Valid: true},
			}
		case model.LocationVariantSolarSystem:
			t, err := s.getOrCreateEveTypeESI(ctx, model.EveTypeIDSolarSystem)
			if err != nil {
				return nil, err
			}
			x, err := s.getOrCreateEveSolarSystemESI(ctx, int32(id))
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:               id,
				EveTypeID:        sql.NullInt32{Int32: t.ID, Valid: true},
				EveSolarSystemID: sql.NullInt32{Int32: x.ID, Valid: true},
			}
		case model.LocationVariantStation:
			station, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStationsStationId(ctx, int32(id), nil)
			if err != nil {
				return nil, err
			}
			_, err = s.getOrCreateEveSolarSystemESI(ctx, station.SystemId)
			if err != nil {
				return nil, err
			}
			_, err = s.getOrCreateEveTypeESI(ctx, station.TypeId)
			if err != nil {
				return nil, err
			}
			arg.EveTypeID = sql.NullInt32{Int32: station.TypeId, Valid: true}
			arg = storage.UpdateOrCreateLocationParams{
				ID:               id,
				EveSolarSystemID: sql.NullInt32{Int32: station.SystemId, Valid: true},
				EveTypeID:        sql.NullInt32{Int32: station.TypeId, Valid: true},
				Name:             station.Name,
			}
			if station.Owner != 0 {
				_, err = s.AddMissingEveEntities(ctx, []int32{station.Owner})
				if err != nil {
					return nil, err
				}
				arg.OwnerID = sql.NullInt32{Int32: station.Owner, Valid: true}
			}

		case model.LocationVariantStructure:
			structure, r, err := s.esiClient.ESI.UniverseApi.GetUniverseStructuresStructureId(ctx, id, nil)
			if err != nil {
				if r != nil && r.StatusCode == http.StatusForbidden {
					arg = storage.UpdateOrCreateLocationParams{ID: id}
					break
				}
				return nil, err
			}
			_, err = s.getOrCreateEveSolarSystemESI(ctx, structure.SolarSystemId)
			if err != nil {
				return nil, err
			}
			_, err = s.AddMissingEveEntities(ctx, []int32{structure.OwnerId})
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:               id,
				EveSolarSystemID: sql.NullInt32{Int32: structure.SolarSystemId, Valid: true},
				Name:             structure.Name,
				OwnerID:          sql.NullInt32{Int32: structure.OwnerId, Valid: true},
			}
			if structure.TypeId != 0 {
				myType, err := s.getOrCreateEveTypeESI(ctx, structure.TypeId)
				if err != nil {
					return nil, err
				}
				arg.EveTypeID = sql.NullInt32{Int32: myType.ID, Valid: true}
			}
		default:
			return nil, fmt.Errorf("can not update or create structure for invalid ID: %d", id)
		}
		if err := s.r.UpdateOrCreateLocation(ctx, arg); err != nil {
			return nil, err
		}
		return s.r.GetLocation(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.Location), nil
}
