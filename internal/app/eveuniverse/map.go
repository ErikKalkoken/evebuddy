package eveuniverse

import (
	"cmp"
	"context"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type Planet struct {
	AsteroidBelts []int32
	Moons         []int32
	ID            int32
}

type SolarSystemPlus struct {
	System   *app.EveSolarSystem
	Stations []*app.EveEntity
	StarID   int32
}

func (s *EveUniverseService) GetOrCreateEveSolarSystemESI2(ctx context.Context, solarSystemID int32) (SolarSystemPlus, error) {
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
	r.StarID = x.StarId
	_, err = s.AddMissingEveEntities(ctx, slices.Concat([]int32{solarSystemID, o.Constellation.ID, o.Constellation.Region.ID}, x.Stations))
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
	return r, nil
}

func (s *EveUniverseService) GetStarTypeESI(ctx context.Context, starID int32) (*app.EveEntity, error) {
	x, _, err := s.esiClient.ESI.UniverseApi.GetUniverseStarsStarId(ctx, starID, nil)
	if err != nil {
		return nil, err
	}
	return s.GetOrCreateEveEntityESI(ctx, x.TypeId)
}
