package eveuniverse

import (
	"cmp"
	"context"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"golang.org/x/sync/errgroup"
)

type SolarSystemPlus struct {
	AdjacentSystems []*app.EveSolarSystem
	System          *app.EveSolarSystem
	Stations        []*app.EveEntity
	Structures      []*app.EveLocation
	StarID          int32
	StarTypeID      int32
}

func (s *EveUniverseService) GetOrCreateEveSolarSystemESIPlus(ctx context.Context, solarSystemID int32) (SolarSystemPlus, error) {
	var r SolarSystemPlus
	o, err := s.GetOrCreateEveSolarSystemESI(ctx, solarSystemID)
	if err != nil {
		return r, err
	}
	r.System = o
	x, _, err := s.esiClient.ESI.UniverseApi.GetUniverseSystemsSystemId(ctx, solarSystemID, nil)
	if err != nil {
		return r, err
	}
	x2, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStarsStarId(ctx, x.StarId, nil)
	if err != nil {
		return r, err
	}
	r.StarTypeID = x2.TypeId

	g := new(errgroup.Group)
	adjacentSystems := make([]int32, len(x.Stargates))
	for i, id := range x.Stargates {
		g.Go(func() error {
			x, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStargatesStargateId(ctx, id, nil)
			if err != nil {
				return err
			}
			adjacentSystems[i] = x.Destination.SystemId
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return r, err
	}

	_, err = s.AddMissingEveEntities(ctx, slices.Concat(
		[]int32{solarSystemID, o.Constellation.ID, o.Constellation.Region.ID},
		x.Stations,
		adjacentSystems,
	))
	if err != nil {
		return r, err
	}

	g = new(errgroup.Group)
	r.AdjacentSystems = make([]*app.EveSolarSystem, len(adjacentSystems))
	for i, id := range adjacentSystems {
		g.Go(func() error {
			st, err := s.GetOrCreateEveSolarSystemESI(ctx, id)
			if err != nil {
				return err
			}
			r.AdjacentSystems[i] = st
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return r, err
	}
	slices.SortFunc(r.AdjacentSystems, func(a, b *app.EveSolarSystem) int {
		return cmp.Compare(a.Name, b.Name)
	})

	r.Stations = make([]*app.EveEntity, len(x.Stations))
	for i, id := range x.Stations {
		st, err := s.getValidEveEntity(ctx, id)
		if err != nil {
			return r, err
		}
		r.Stations[i] = st
	}
	slices.SortFunc(r.Stations, func(a, b *app.EveEntity) int {
		return cmp.Compare(a.Name, b.Name)
	})
	xx, err := s.st.ListEveLocationInSolarSystem(ctx, solarSystemID)
	if err != nil {
		return r, err
	}
	r.Structures = slices.Collect(xiter.FilterSlice(xx, func(x *app.EveLocation) bool {
		return x.Variant() == app.EveLocationStructure
	}))
	return r, nil
}
