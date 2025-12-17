package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/ErikKalkoken/kx/set"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func (s *EveUniverseService) GetLocation(ctx context.Context, id int64) (*app.EveLocation, error) {
	return s.st.GetLocation(ctx, id)
}

func (s *EveUniverseService) ListLocations(ctx context.Context) ([]*app.EveLocation, error) {
	return s.st.ListEveLocation(ctx)
}

// GetOrCreateLocationESI return a location.
// When the location does not yet exist in storage it tries to fetch and create a new location from ESI.
//
// Important: For creating structures a valid token with the structure scope must be set in the context or an error will be returned
func (s *EveUniverseService) GetOrCreateLocationESI(ctx context.Context, id int64) (*app.EveLocation, error) {
	o, err := s.st.GetLocation(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return s.UpdateOrCreateLocationESI(ctx, id)
	}
	return o, err
}

// UpdateOrCreateLocationESI tries to fetch and create a new location from ESI.
//
// Important: For creating structures a valid token with the structure scope must be set in the context or an error will be returned
func (s *EveUniverseService) UpdateOrCreateLocationESI(ctx context.Context, id int64) (*app.EveLocation, error) {
	y, err, _ := s.sfg.Do(fmt.Sprintf("updateOrCreateLocationESI-%d", id), func() (any, error) {
		o, err := s.st.GetLocation(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		var arg storage.UpdateOrCreateLocationParams
		switch app.LocationVariantFromID(id) {
		case app.EveLocationUnknown:
			t, err := s.GetOrCreateTypeESI(ctx, app.EveTypeSolarSystem)
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:     id,
				TypeID: optional.New(t.ID),
			}
		case app.EveLocationAssetSafety:
			t, err := s.GetOrCreateTypeESI(ctx, app.EveTypeAssetSafetyWrap)
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:     id,
				TypeID: optional.New(t.ID),
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
				ID:            id,
				TypeID:        optional.New(et.ID),
				SolarSystemID: optional.New(es.ID),
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
			arg.TypeID = optional.New(station.TypeId)
			arg = storage.UpdateOrCreateLocationParams{
				ID:            id,
				SolarSystemID: optional.New(station.SystemId),
				TypeID:        optional.New(station.TypeId),
				Name:          station.Name,
			}
			if station.Owner != 0 {
				_, err = s.AddMissingEntities(ctx, set.Of(station.Owner))
				if err != nil {
					return nil, err
				}
				arg.OwnerID = optional.New(station.Owner)
			}
		case app.EveLocationStructure:
			if !xgoesi.ContextHasAccessToken(ctx) {
				return nil, fmt.Errorf("eve location: token not set for fetching structure: %d", id)
			}
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetUniverseStructuresStructureId")
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
			_, err = s.AddMissingEntities(ctx, set.Of(structure.OwnerId))
			if err != nil {
				return nil, err
			}
			arg = storage.UpdateOrCreateLocationParams{
				ID:            id,
				SolarSystemID: optional.New(structure.SolarSystemId),
				Name:          structure.Name,
				OwnerID:       optional.New(structure.OwnerId),
			}
			if structure.TypeId != 0 {
				myType, err := s.GetOrCreateTypeESI(ctx, structure.TypeId)
				if err != nil {
					return nil, err
				}
				arg.TypeID = optional.New(myType.ID)
			}
		default:
			return nil, fmt.Errorf("eve location: invalid ID in update or create: %d", id)
		}
		arg.UpdatedAt = time.Now()
		if err := s.st.UpdateOrCreateEveLocation(ctx, arg); err != nil {
			return nil, err
		}
		slog.Info("Stored updated eve location", "ID", id)
		return s.st.GetLocation(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveLocation), nil
}

// AddMissingLocations adds missing EveLocations from ESI.
// Invalid IDs (e.g. ID 0) will be ignored.
func (s *EveUniverseService) AddMissingLocations(ctx context.Context, ids set.Set[int64]) error {
	ids2 := ids.Clone()
	ids2.Delete(0)
	if ids2.Size() == 0 {
		return nil
	}
	missing, err := s.st.MissingEveLocations(ctx, ids2)
	if err != nil {
		return err
	}
	if missing.Size() == 0 {
		return nil
	}
	entities, err := s.EntityIDsFromLocationsESI(ctx, missing.Slice())
	if err != nil {
		return err
	}
	if _, err := s.AddMissingEntities(ctx, entities); err != nil {
		return err
	}
	g := new(errgroup.Group)
	g.SetLimit(s.concurrencyLimit)
	for id := range missing.All() {
		g.Go(func() error {
			_, err := s.GetOrCreateLocationESI(ctx, id)
			return err
		})
	}
	return g.Wait()
}

// EntityIDsFromLocationsESI returns the EveEntity IDs in EveLocation ids from ESI.
// This methods allows bulk resolving EveEntities before fetching many new locations from ESI.
func (s *EveUniverseService) EntityIDsFromLocationsESI(ctx context.Context, ids []int64) (set.Set[int32], error) {
	if len(ids) == 0 {
		return set.Set[int32]{}, nil
	}
	for _, id := range ids {
		if app.LocationVariantFromID(id) == app.EveLocationStructure {
			if !xgoesi.ContextHasAccessToken(ctx) {
				return set.Set[int32]{}, fmt.Errorf("EntityIDsFromLocationsESI: token not set for location ID %d: %w", id, app.ErrInvalid)
			}
			break
		}
	}
	entityIDs := make([]int32, len(ids))
	g := new(errgroup.Group)
	g.SetLimit(s.concurrencyLimit)
	for i, id := range ids {
		g.Go(func() error {
			switch app.LocationVariantFromID(id) {
			case app.EveLocationStation:
				station, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStationsStationId(ctx, int32(id), nil)
				if err != nil {
					return err
				}
				if x := station.Owner; x != 0 {
					entityIDs[i] = x
				}
			case app.EveLocationStructure:
				ctx = xgoesi.NewContextWithOperationID(ctx, "GetUniverseStructuresStructureId")
				structure, r, err := s.esiClient.ESI.UniverseApi.GetUniverseStructuresStructureId(ctx, id, nil)
				if err != nil {
					if r != nil && r.StatusCode == http.StatusForbidden {
						return nil
					}
					return err
				}
				entityIDs[i] = structure.OwnerId
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return set.Set[int32]{}, err
	}
	r := set.Of(xslices.Filter(entityIDs, func(x int32) bool {
		return x != 0 && x != 1 && x != -1
	})...)
	return r, nil
}

// GetStationServicesESI fetches and returns the services of a station from ESI.
func (s *EveUniverseService) GetStationServicesESI(ctx context.Context, id int32) ([]string, error) {
	o, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStationsStationId(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	slices.Sort(o.Services)
	return o.Services, nil
}
