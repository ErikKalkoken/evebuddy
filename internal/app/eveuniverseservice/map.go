package eveuniverseservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

// FetchRoute fetches a route between two solar systems from ESi and returns it.
// When no route can be found it returns an empty slice.
func (s *EveUniverseService) FetchRoute(ctx context.Context, args app.EveRouteHeader) ([]*app.EveSolarSystem, error) {
	m := map[app.EveRoutePreference]string{
		app.RouteShorter:    "Shorter",
		app.RouteSafer:      "Safer",
		app.RouteLessSecure: "LessSecure",
	}
	flag, ok := m[args.Preference]
	if !ok {
		return nil, fmt.Errorf("FetchRoute: flag %s: %w", args.Preference, app.ErrInvalid)
	}
	if args.Destination == nil || args.Origin == nil {
		return nil, app.ErrInvalid
	}
	if args.Destination.ID == args.Origin.ID {
		return []*app.EveSolarSystem{args.Origin}, nil
	}
	if args.Destination.IsWormholeSpace() || args.Origin.IsWormholeSpace() {
		return []*app.EveSolarSystem{}, nil // no route possible
	}
	arg := esi.RouteRequestBody{
		Preference: &flag,
	}
	route, r, err := s.esiClient.RoutesAPI.PostRoute(ctx, args.Origin.ID, args.Destination.ID).RouteRequestBody(arg).Execute()
	if err != nil {
		if r != nil && r.StatusCode == 404 {
			return []*app.EveSolarSystem{}, nil // no route found
		}
		return nil, err
	}
	systems := make([]*app.EveSolarSystem, len(route.Route))
	g := new(errgroup.Group)
	g.SetLimit(s.concurrencyLimit)
	for i, id := range route.Route {
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

// FetchRoutes returns routes for one or multiple headers.
func (s *EveUniverseService) FetchRoutes(ctx context.Context, headers []app.EveRouteHeader) (map[app.EveRouteHeader][]*app.EveSolarSystem, error) {
	results := make([][]*app.EveSolarSystem, len(headers))
	g := new(errgroup.Group)
	g.SetLimit(s.concurrencyLimit)
	for i, h := range headers {
		g.Go(func() error {
			route, err := s.FetchRoute(ctx, h)
			if err != nil {
				return err
			}
			results[i] = route
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	results2 := make(map[app.EveRouteHeader][]*app.EveSolarSystem)
	for i, h := range headers {
		results2[h] = results[i]
	}
	return results2, nil
}

// GetStargatesSolarSystemsESI fetches and returns the solar systems which relates to given stargates from ESI.
func (s *EveUniverseService) GetStargatesSolarSystemsESI(ctx context.Context, stargateIDs []int64) ([]*app.EveSolarSystem, error) {
	g := new(errgroup.Group)
	g.SetLimit(s.concurrencyLimit)
	systemIDs := make([]int64, len(stargateIDs))
	for i, id := range stargateIDs {
		g.Go(func() error {
			x, _, err := s.esiClient.UniverseAPI.GetUniverseStargatesStargateId(ctx, id).Execute()
			if err != nil {
				return err
			}
			systemIDs[i] = x.Destination.SystemId
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	g = new(errgroup.Group)
	g.SetLimit(s.concurrencyLimit)
	systems := make([]*app.EveSolarSystem, len(systemIDs))
	for i, id := range systemIDs {
		g.Go(func() error {
			st, err := s.GetOrCreateSolarSystemESI(ctx, id)
			if err != nil {
				return err
			}
			systems[i] = st
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	slices.SortFunc(systems, func(a, b *app.EveSolarSystem) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return systems, nil
}

// GetSolarSystemPlanets fetches and returns the planets for a solar system from ESI.
func (s *EveUniverseService) GetSolarSystemPlanets(ctx context.Context, planets []app.EveSolarSystemPlanet) ([]*app.EvePlanet, error) {
	oo := make([]*app.EvePlanet, len(planets))
	g := new(errgroup.Group)
	g.SetLimit(s.concurrencyLimit)
	for i, p := range planets {
		g.Go(func() error {
			st, err := s.GetOrCreatePlanetESI(ctx, p.PlanetID)
			if err != nil {
				return err
			}
			oo[i] = st
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	slices.SortFunc(oo, func(a, b *app.EvePlanet) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return oo, nil
}

// GetSolarSystemInfoESI fetches and returns details about a solar system from ESI.
func (s *EveUniverseService) GetSolarSystemInfoESI(ctx context.Context, solarSystemID int64) (starID optional.Optional[int64], planets []app.EveSolarSystemPlanet, stargateIDs []int64, stations []*app.EveEntity, structures []*app.EveLocation, err error) {
	var z optional.Optional[int64]
	system, _, err := s.esiClient.UniverseAPI.GetUniverseSystemsSystemId(ctx, solarSystemID).Execute()
	if err != nil {
		return z, nil, nil, nil, nil, err
	}
	planets = xslices.Map(system.Planets, func(p esi.UniverseSystemsSystemIdGetPlanetsInner) app.EveSolarSystemPlanet {
		return app.EveSolarSystemPlanet{
			AsteroidBeltIDs: p.AsteroidBelts,
			MoonIDs:         p.Moons,
			PlanetID:        p.PlanetId,
		}
	})
	ids := slices.Concat([]int64{solarSystemID, system.ConstellationId}, system.Stations)
	_, err = s.AddMissingEntities(ctx, set.Of(ids...))
	if err != nil {
		return z, nil, nil, nil, nil, err
	}
	stations = make([]*app.EveEntity, len(system.Stations))
	for i, id := range system.Stations {
		st, err := s.getValidEntity(ctx, id)
		if err != nil {
			return z, nil, nil, nil, nil, err
		}
		stations[i] = st
	}
	slices.SortFunc(stations, func(a, b *app.EveEntity) int {
		return a.Compare(b)
	})
	xx, err := s.st.ListEveLocationInSolarSystem(ctx, solarSystemID)
	if err != nil {
		return z, nil, nil, nil, nil, err
	}
	structures = xslices.Filter(xx, func(x *app.EveLocation) bool {
		return x.Variant() == app.EveLocationStructure
	})
	return optional.FromPtr(system.StarId), planets, system.Stargates, stations, structures, nil
}

// GetRegionConstellationsESI fetches and returns the constellations for a region.
func (s *EveUniverseService) GetRegionConstellationsESI(ctx context.Context, id int64) ([]*app.EveEntity, error) {
	region, _, err := s.esiClient.UniverseAPI.GetUniverseRegionsRegionId(ctx, id).Execute()
	if err != nil {
		return nil, err
	}
	ee, err := s.ToEntities(ctx, set.Of(region.Constellations...))
	if err != nil {
		return nil, err
	}
	oo := slices.Collect(maps.Values(ee))
	slices.SortFunc(oo, func(a, b *app.EveEntity) int {
		return a.Compare(b)
	})
	return oo, nil
}

// GetConstellationSolarSystemsESI fetches and returns the solar systems for a constellations from ESI.
func (s *EveUniverseService) GetConstellationSolarSystemsESI(ctx context.Context, id int64) ([]*app.EveSolarSystem, error) {
	o, _, err := s.esiClient.UniverseAPI.GetUniverseConstellationsConstellationId(ctx, id).Execute()
	if err != nil {
		return nil, err
	}
	g := new(errgroup.Group)
	g.SetLimit(s.concurrencyLimit)
	systems := make([]*app.EveSolarSystem, len(o.Systems))
	for i, id := range o.Systems {
		g.Go(func() error {
			st, err := s.GetOrCreateSolarSystemESI(ctx, id)
			if err != nil {
				return err
			}
			systems[i] = st
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	slices.SortFunc(systems, func(a, b *app.EveSolarSystem) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return systems, nil
}

func (s *EveUniverseService) GetOrCreateRegionESI(ctx context.Context, id int64) (*app.EveRegion, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateRegionESI-%d", id), func() (any, error) {
		o1, err := s.st.GetEveRegion(ctx, id)
		if err == nil {
			return o1, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		region, _, err := s.esiClient.UniverseAPI.GetUniverseRegionsRegionId(ctx, id).Execute()
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveRegionParams{
			ID:          region.RegionId,
			Description: optional.FromPtr(region.Description),
			Name:        region.Name,
		}
		o2, err := s.st.CreateEveRegion(ctx, arg)
		if err != nil {
			return nil, err
		}
		slog.Info("Created eve region", "ID", id)
		return o2, nil
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveRegion), nil
}

func (s *EveUniverseService) GetOrCreateConstellationESI(ctx context.Context, id int64) (*app.EveConstellation, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateConstellationESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveConstellation(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		constellation, _, err := s.esiClient.UniverseAPI.GetUniverseConstellationsConstellationId(ctx, id).Execute()
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
		slog.Info("Created eve constellation", "ID", id)
		return s.st.GetEveConstellation(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveConstellation), nil
}

func (s *EveUniverseService) GetOrCreateSolarSystemESI(ctx context.Context, id int64) (*app.EveSolarSystem, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateSolarSystemESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveSolarSystem(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		system, _, err := s.esiClient.UniverseAPI.GetUniverseSystemsSystemId(ctx, id).Execute()
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
		slog.Info("Created eve solar system", "ID", id)
		return s.st.GetEveSolarSystem(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveSolarSystem), nil
}

func (s *EveUniverseService) GetOrCreatePlanetESI(ctx context.Context, id int64) (*app.EvePlanet, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreatePlanetESI-%d", id), func() (any, error) {
		o, err := s.st.GetEvePlanet(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		planet, _, err := s.esiClient.UniverseAPI.GetUniversePlanetsPlanetId(ctx, id).Execute()
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
		slog.Info("Created eve planet", "ID", id)
		return s.st.GetEvePlanet(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EvePlanet), nil
}

func (s *EveUniverseService) GetOrCreateMoonESI(ctx context.Context, id int64) (*app.EveMoon, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateMoonESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveMoon(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		moon, _, err := s.esiClient.UniverseAPI.GetUniverseMoonsMoonId(ctx, id).Execute()
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
		slog.Info("Created eve moon", "ID", id)
		return s.st.GetEveMoon(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveMoon), nil
}

func (s *EveUniverseService) GetStarTypeID(ctx context.Context, id int64) (int64, error) {
	x2, _, err := s.esiClient.UniverseAPI.GetUniverseStarsStarId(ctx, id).Execute()
	if err != nil {
		return 0, err
	}
	return x2.TypeId, nil
}

// AddMissingRegions fetches missing regions from ESI.
// Invalid IDs (e.g. 0) will be ignored
func (s *EveUniverseService) AddMissingRegions(ctx context.Context, ids set.Set[int64]) error {
	ids2 := ids.Clone()
	ids2.Delete(0) // ignore invalid ID
	if ids.Size() == 0 {
		return nil
	}
	missing, err := s.st.MissingEveRegions(ctx, ids2)
	if err != nil {
		return err
	}
	if missing.Size() == 0 {
		return nil
	}
	slog.Debug("Trying to fetch missing regions from ESI", "count", missing.Size())
	g := new(errgroup.Group)
	g.SetLimit(s.concurrencyLimit)
	for id := range missing.All() {
		g.Go(func() error {
			_, err := s.GetOrCreateRegionESI(ctx, id)
			return err
		})
	}
	return g.Wait()
}

// AddMissingSolarSystems fetches missing solar systems from ESI.
// Invalid IDs (e.g. 0) will be ignored
func (s *EveUniverseService) AddMissingSolarSystems(ctx context.Context, ids set.Set[int64]) error {
	ids2 := ids.Clone()
	ids2.Delete(0) // ignore invalid ID
	if ids.Size() == 0 {
		return nil
	}
	missing, err := s.st.MissingEveSolarSystems(ctx, ids2)
	if err != nil {
		return err
	}
	if missing.Size() == 0 {
		return nil
	}
	slog.Debug("Trying to fetch missing solar systems from ESI", "count", missing.Size())
	g := new(errgroup.Group)
	g.SetLimit(s.concurrencyLimit)
	for id := range missing.All() {
		g.Go(func() error {
			_, err := s.GetOrCreateSolarSystemESI(ctx, id)
			return err
		})
	}
	return g.Wait()
}
