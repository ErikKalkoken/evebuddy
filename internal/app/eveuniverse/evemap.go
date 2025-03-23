package eveuniverse

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

func (s *EveUniverseService) GetOrCreateRegionESI(ctx context.Context, id int32) (*app.EveRegion, error) {
	x, err := s.st.GetEveRegion(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return s.createEveRegionFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (s *EveUniverseService) createEveRegionFromESI(ctx context.Context, id int32) (*app.EveRegion, error) {
	key := fmt.Sprintf("createEveRegionFromESI-%d", id)
	y, err, _ := s.sfg.Do(key, func() (any, error) {
		region, _, err := s.esiClient.ESI.UniverseApi.GetUniverseRegionsRegionId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveRegionParams{
			ID:          region.RegionId,
			Description: region.Description,
			Name:        region.Name,
		}
		return s.st.CreateEveRegion(ctx, arg)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveRegion), nil
}

func (s *EveUniverseService) GetOrCreateConstellationESI(ctx context.Context, id int32) (*app.EveConstellation, error) {
	x, err := s.st.GetEveConstellation(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return s.createEveConstellationFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (s *EveUniverseService) createEveConstellationFromESI(ctx context.Context, id int32) (*app.EveConstellation, error) {
	key := fmt.Sprintf("createEveConstellationFromESI-%d", id)
	y, err, _ := s.sfg.Do(key, func() (any, error) {
		constellation, _, err := s.esiClient.ESI.UniverseApi.GetUniverseConstellationsConstellationId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		_, err = s.GetOrCreateRegionESI(ctx, constellation.RegionId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveConstellationParams{
			ID:       constellation.ConstellationId,
			RegionID: constellation.RegionId,
			Name:     constellation.Name,
		}
		if err := s.st.CreateEveConstellation(ctx, arg); err != nil {
			return nil, err
		}
		return s.st.GetEveConstellation(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveConstellation), nil
}

func (s *EveUniverseService) GetOrCreateSolarSystemESI(ctx context.Context, id int32) (*app.EveSolarSystem, error) {
	x, err := s.st.GetEveSolarSystem(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return s.createEveSolarSystemFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (s *EveUniverseService) createEveSolarSystemFromESI(ctx context.Context, id int32) (*app.EveSolarSystem, error) {
	key := fmt.Sprintf("createEveSolarSystemFromESI-%d", id)
	y, err, _ := s.sfg.Do(key, func() (any, error) {
		system, _, err := s.esiClient.ESI.UniverseApi.GetUniverseSystemsSystemId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		constellation, err := s.GetOrCreateConstellationESI(ctx, system.ConstellationId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveSolarSystemParams{
			ID:              system.SystemId,
			ConstellationID: constellation.ID,
			Name:            system.Name,
			SecurityStatus:  system.SecurityStatus,
		}
		if err := s.st.CreateEveSolarSystem(ctx, arg); err != nil {
			return nil, err
		}
		return s.st.GetEveSolarSystem(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveSolarSystem), nil
}

func (s *EveUniverseService) GetOrCreatePlanetESI(ctx context.Context, id int32) (*app.EvePlanet, error) {
	x, err := s.st.GetEvePlanet(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return s.createEvePlanetFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (s *EveUniverseService) createEvePlanetFromESI(ctx context.Context, id int32) (*app.EvePlanet, error) {
	key := fmt.Sprintf("createEvePlanetFromESI-%d", id)
	y, err, _ := s.sfg.Do(key, func() (any, error) {
		planet, _, err := s.esiClient.ESI.UniverseApi.GetUniversePlanetsPlanetId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		system, err := s.GetOrCreateSolarSystemESI(ctx, planet.SystemId)
		if err != nil {
			return nil, err
		}
		type_, err := s.GetOrCreateTypeESI(ctx, planet.TypeId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEvePlanetParams{
			ID:            planet.PlanetId,
			Name:          planet.Name,
			SolarSystemID: system.ID,
			TypeID:        type_.ID,
		}
		if err := s.st.CreateEvePlanet(ctx, arg); err != nil {
			return nil, err
		}
		return s.st.GetEvePlanet(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EvePlanet), nil
}

func (s *EveUniverseService) GetOrCreateEveMoonESI(ctx context.Context, id int32) (*app.EveMoon, error) {
	x, err := s.st.GetEveMoon(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return s.createEveMoonFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (s *EveUniverseService) createEveMoonFromESI(ctx context.Context, id int32) (*app.EveMoon, error) {
	key := fmt.Sprintf("createEveMoonFromESI-%d", id)
	y, err, _ := s.sfg.Do(key, func() (any, error) {
		moon, _, err := s.esiClient.ESI.UniverseApi.GetUniverseMoonsMoonId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		system, err := s.GetOrCreateSolarSystemESI(ctx, moon.SystemId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveMoonParams{
			ID:            moon.MoonId,
			Name:          moon.Name,
			SolarSystemID: system.ID,
		}
		if err := s.st.CreateEveMoon(ctx, arg); err != nil {
			return nil, err
		}
		return s.st.GetEveMoon(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveMoon), nil
}

// GetRouteESI returns a route between two solar systems.
// When no route can be found it returns an empty slice.
type RoutePreference string

func (s *EveUniverseService) GetRouteESI(ctx context.Context, destination, origin *app.EveSolarSystem, flag app.RoutePreference) ([]*app.EveSolarSystem, error) {
	if slices.Index(app.RoutePreferences(), flag) == -1 {
		return nil, fmt.Errorf("invalid flag: %s", flag)
	}
	if destination.ID == origin.ID {
		return []*app.EveSolarSystem{origin}, nil
	}
	if destination.IsWormholeSpace() || origin.IsWormholeSpace() {
		return []*app.EveSolarSystem{}, nil // no route possible
	}
	ids, r, err := s.esiClient.ESI.RoutesApi.GetRouteOriginDestination(ctx, destination.ID, origin.ID, &esi.GetRouteOriginDestinationOpts{
		Flag: esioptional.NewString(flag.String()),
	})
	if err != nil {
		if r.StatusCode == 404 {
			return []*app.EveSolarSystem{}, nil // no route found
		}
		return nil, err
	}
	systems := make([]*app.EveSolarSystem, len(ids))
	g := new(errgroup.Group)
	for i, id := range ids {
		g.Go(func() error {
			system, err := s.GetOrCreateSolarSystemESI(ctx, id)
			if err != nil {
				return err
			}
			systems[i] = system
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return systems, nil
}
