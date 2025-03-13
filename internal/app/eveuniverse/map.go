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
	System     *app.EveSolarSystem
	Stations   []*app.EveEntity
	Structures []*app.EveLocation
	StarID     int32

	stargates []int32
	eus       *EveUniverseService
}

func (o SolarSystemPlus) GetAdjacentSystems(ctx context.Context) ([]*app.EveSolarSystem, error) {
	g := new(errgroup.Group)
	systemIDs := make([]int32, len(o.stargates))
	for i, id := range o.stargates {
		g.Go(func() error {
			x, _, err := o.eus.esiClient.ESI.UniverseApi.GetUniverseStargatesStargateId(ctx, id, nil)
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
			st, err := o.eus.GetOrCreateEveSolarSystemESI(ctx, id)
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

func (o SolarSystemPlus) GetStarTypeID(ctx context.Context) (int32, error) {
	x2, _, err := o.eus.esiClient.ESI.UniverseApi.GetUniverseStarsStarId(ctx, o.StarID, nil)
	if err != nil {
		return 0, err
	}
	return x2.TypeId, nil
}

func (s *EveUniverseService) GetOrCreateEveSolarSystemESIPlus(ctx context.Context, solarSystemID int32) (SolarSystemPlus, error) {
	r := SolarSystemPlus{eus: s}
	o, err := s.GetOrCreateEveSolarSystemESI(ctx, solarSystemID)
	if err != nil {
		return SolarSystemPlus{}, err
	}
	r.System = o
	x, _, err := s.esiClient.ESI.UniverseApi.GetUniverseSystemsSystemId(ctx, solarSystemID, nil)
	if err != nil {
		return r, err
	}
	r.stargates = x.Stargates
	r.StarID = x.StarId
	_, err = s.AddMissingEveEntities(ctx, slices.Concat(
		[]int32{solarSystemID, o.Constellation.ID, o.Constellation.Region.ID},
		x.Stations,
	))
	if err != nil {
		return r, err
	}
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
