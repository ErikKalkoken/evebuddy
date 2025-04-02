package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi"
)

func (s *EveUniverseService) GetStationServicesESI(ctx context.Context, id int32) ([]string, error) {
	o, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStationsStationId(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	slices.Sort(o.Services)
	return o.Services, nil
}

func (s *EveUniverseService) GetLocation(ctx context.Context, id int64) (*app.EveLocation, error) {
	o, err := s.st.GetLocation(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return nil, app.ErrNotFound
	}
	return o, err
}

func (s *EveUniverseService) ListLocations(ctx context.Context) ([]*app.EveLocation, error) {
	return s.st.ListEveLocation(ctx)
}

// GetOrCreateLocationESI return a structure when it already exists
// or else tries to fetch and create a new structure from ESI.
//
// Important: A token with the structure scope must be set in the context
func (s *EveUniverseService) GetOrCreateLocationESI(ctx context.Context, id int64) (*app.EveLocation, error) {
	o, err := s.st.GetLocation(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return s.updateOrCreateLocationESI(ctx, id)
	}
	return o, err
}

// updateOrCreateLocationESI tries to fetch and create a new structure from ESI.
//
// Important: A token with the structure scope must be set in the context when trying to fetch a structure.
func (s *EveUniverseService) updateOrCreateLocationESI(ctx context.Context, id int64) (*app.EveLocation, error) {
	key := fmt.Sprintf("updateOrCreateLocationESI-%d", id)
	y, err, _ := s.sfg.Do(key, func() (any, error) {
		var arg storage.UpdateOrCreateLocationParams
		switch app.LocationVariantFromID(id) {
		case app.EveLocationUnknown:
			t, err := s.GetOrCreateTypeESI(ctx, app.EveTypeSolarSystem)
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:        id,
				EveTypeID: optional.New(t.ID),
			}
		case app.EveLocationAssetSafety:
			t, err := s.GetOrCreateTypeESI(ctx, app.EveTypeAssetSafetyWrap)
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:        id,
				EveTypeID: optional.New(t.ID),
			}
		case app.EveLocationSolarSystem:
			et, err := s.GetOrCreateTypeESI(ctx, app.EveTypeSolarSystem)
			if err != nil {
				return nil, err
			}
			es, err := s.GetOrCreateSolarSystemESI(ctx, int32(id))
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:               id,
				EveTypeID:        optional.New(et.ID),
				EveSolarSystemID: optional.New(es.ID),
			}
		case app.EveLocationStation:
			station, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStationsStationId(ctx, int32(id), nil)
			if err != nil {
				return nil, err
			}
			_, err = s.GetOrCreateSolarSystemESI(ctx, station.SystemId)
			if err != nil {
				return nil, err
			}
			_, err = s.GetOrCreateTypeESI(ctx, station.TypeId)
			if err != nil {
				return nil, err
			}
			arg.EveTypeID = optional.New(station.TypeId)
			arg = storage.UpdateOrCreateLocationParams{
				ID:               id,
				EveSolarSystemID: optional.New(station.SystemId),
				EveTypeID:        optional.New(station.TypeId),
				Name:             station.Name,
			}
			if station.Owner != 0 {
				_, err = s.AddMissingEntities(ctx, []int32{station.Owner})
				if err != nil {
					return nil, err
				}
				arg.OwnerID = optional.New(station.Owner)
			}
		case app.EveLocationStructure:
			if ctx.Value(goesi.ContextAccessToken) == nil {
				return nil, fmt.Errorf("eve location: token not set for fetching structure: %d", id)
			}
			structure, r, err := s.esiClient.ESI.UniverseApi.GetUniverseStructuresStructureId(ctx, id, nil)
			if err != nil {
				if r != nil && r.StatusCode == http.StatusForbidden {
					arg = storage.UpdateOrCreateLocationParams{ID: id}
					break
				}
				return nil, err
			}
			_, err = s.GetOrCreateSolarSystemESI(ctx, structure.SolarSystemId)
			if err != nil {
				return nil, err
			}
			_, err = s.AddMissingEntities(ctx, []int32{structure.OwnerId})
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:               id,
				EveSolarSystemID: optional.New(structure.SolarSystemId),
				Name:             structure.Name,
				OwnerID:          optional.New(structure.OwnerId),
			}
			if structure.TypeId != 0 {
				myType, err := s.GetOrCreateTypeESI(ctx, structure.TypeId)
				if err != nil {
					return nil, err
				}
				arg.EveTypeID = optional.New(myType.ID)
			}
		default:
			return nil, fmt.Errorf("eve location: invalid ID in update or create: %d", id)
		}
		arg.UpdatedAt = time.Now()
		if err := s.st.UpdateOrCreateEveLocation(ctx, arg); err != nil {
			return nil, err
		}
		return s.st.GetLocation(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveLocation), nil
}
