package eveuniverseservice

import (
	"cmp"
	"context"
	"maps"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/antihax/goesi/esi"
	"golang.org/x/sync/errgroup"
)

// type SolarSystemPlus struct {
// 	System     *app.EveSolarSystem
// 	Stations   []*app.EveEntity
// 	Structures []*app.EveLocation
// 	StarID     int32

// 	planets   []planet
// 	stargates []int32
// 	eus       *EveUniverseService
// }

func (s *EveUniverseService) GetSolarSystemsESI(ctx context.Context, stargateIDs []int32) ([]*app.EveSolarSystem, error) {
	g := new(errgroup.Group)
	systemIDs := make([]int32, len(stargateIDs))
	for i, id := range stargateIDs {
		g.Go(func() error {
			x, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStargatesStargateId(ctx, id, nil)
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

func (s *EveUniverseService) GetPlanets(ctx context.Context, planets []app.EveSolarSystemPlanet) ([]*app.EvePlanet, error) {
	oo := make([]*app.EvePlanet, len(planets))
	g := new(errgroup.Group)
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

func (s *EveUniverseService) GetStarTypeID(ctx context.Context, id int32) (int32, error) {
	x2, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStarsStarId(ctx, id, nil)
	if err != nil {
		return 0, err
	}
	return x2.TypeId, nil
}

func (s *EveUniverseService) GetSolarSystemInfoESI(ctx context.Context, solarSystemID int32) (int32, []app.EveSolarSystemPlanet, []int32, []*app.EveEntity, []*app.EveLocation, error) {
	x, _, err := s.esiClient.ESI.UniverseApi.GetUniverseSystemsSystemId(ctx, solarSystemID, nil)
	if err != nil {
		return 0, nil, nil, nil, nil, err
	}
	planets := xslices.Map(x.Planets, func(p esi.GetUniverseSystemsSystemIdPlanet) app.EveSolarSystemPlanet {
		return app.EveSolarSystemPlanet{
			AsteroidBeltIDs: p.AsteroidBelts,
			MoonIDs:         p.Moons,
			PlanetID:        p.PlanetId,
		}
	})
	_, err = s.AddMissingEntities(ctx, slices.Concat(
		[]int32{solarSystemID, x.ConstellationId},
		x.Stations,
	))
	if err != nil {
		return 0, nil, nil, nil, nil, err
	}
	stations := make([]*app.EveEntity, len(x.Stations))
	for i, id := range x.Stations {
		st, err := s.getValidEveEntity(ctx, id)
		if err != nil {
			return 0, nil, nil, nil, nil, err
		}
		stations[i] = st
	}
	slices.SortFunc(stations, func(a, b *app.EveEntity) int {
		return a.Compare(b)
	})
	xx, err := s.st.ListEveLocationInSolarSystem(ctx, solarSystemID)
	if err != nil {
		return 0, nil, nil, nil, nil, err
	}
	structures := xslices.Filter(xx, func(x *app.EveLocation) bool {
		return x.Variant() == app.EveLocationStructure
	})
	return x.StarId, planets, x.Stargates, stations, structures, nil
}

func (s *EveUniverseService) GetRegionConstellationsESI(ctx context.Context, id int32) ([]*app.EveEntity, error) {
	region, _, err := s.esiClient.ESI.UniverseApi.GetUniverseRegionsRegionId(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	xx, err := s.ToEveEntities(ctx, region.Constellations)
	if err != nil {
		return nil, err
	}
	oo := slices.Collect(maps.Values(xx))
	slices.SortFunc(oo, func(a, b *app.EveEntity) int {
		return a.Compare(b)
	})
	return oo, nil
}

func (s *EveUniverseService) GetConstellationSolarSytemsESI(ctx context.Context, id int32) ([]*app.EveSolarSystem, error) {
	o, _, err := s.esiClient.ESI.UniverseApi.GetUniverseConstellationsConstellationId(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	g := new(errgroup.Group)
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
