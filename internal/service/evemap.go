package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (s *Service) GetOrCreateEveRegionESI(id int32) (*model.EveRegion, error) {
	ctx := context.Background()
	return s.getOrCreateEveRegionESI(ctx, id)
}

func (s *Service) getOrCreateEveRegionESI(ctx context.Context, id int32) (*model.EveRegion, error) {
	x, err := s.r.GetEveRegion(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.createEveRegionFromESI(ctx, id)
		}
		return x, err
	}
	return x, nil
}

func (s *Service) createEveRegionFromESI(ctx context.Context, id int32) (*model.EveRegion, error) {
	key := fmt.Sprintf("createEveRegionFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		region, _, err := s.esiClient.ESI.UniverseApi.GetUniverseRegionsRegionId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveRegionParams{
			ID:          region.RegionId,
			Description: region.Description,
			Name:        region.Name,
		}
		return s.r.CreateEveRegion(ctx, arg)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveRegion), nil
}

func (s *Service) GetOrCreateEveConstellationESI(id int32) (*model.EveConstellation, error) {
	ctx := context.Background()
	return s.getOrCreateEveConstellationESI(ctx, id)
}

func (s *Service) getOrCreateEveConstellationESI(ctx context.Context, id int32) (*model.EveConstellation, error) {
	x, err := s.r.GetEveConstellation(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.createEveConstellationFromESI(ctx, id)
		}
		return x, err
	}
	return x, nil
}

func (s *Service) createEveConstellationFromESI(ctx context.Context, id int32) (*model.EveConstellation, error) {
	key := fmt.Sprintf("createEveConstellationFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		constellation, _, err := s.esiClient.ESI.UniverseApi.GetUniverseConstellationsConstellationId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		_, err = s.getOrCreateEveRegionESI(ctx, constellation.RegionId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveConstellationParams{
			ID:       constellation.ConstellationId,
			RegionID: constellation.RegionId,
			Name:     constellation.Name,
		}
		if err := s.r.CreateEveConstellation(ctx, arg); err != nil {
			return nil, err
		}
		return s.r.GetEveConstellation(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveConstellation), nil
}

func (s *Service) GetOrCreateEveSolarSystemESI(id int32) (*model.EveSolarSystem, error) {
	ctx := context.Background()
	return s.getOrCreateEveSolarSystemESI(ctx, id)
}

func (s *Service) getOrCreateEveSolarSystemESI(ctx context.Context, id int32) (*model.EveSolarSystem, error) {
	x, err := s.r.GetEveSolarSystem(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.createEveSolarSystemFromESI(ctx, id)
		}
		return x, err
	}
	return x, nil
}

func (s *Service) createEveSolarSystemFromESI(ctx context.Context, id int32) (*model.EveSolarSystem, error) {
	key := fmt.Sprintf("createEveSolarSystemFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		system, _, err := s.esiClient.ESI.UniverseApi.GetUniverseSystemsSystemId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		constellation, err := s.getOrCreateEveConstellationESI(ctx, system.ConstellationId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveSolarSystemParams{
			ID:              system.SystemId,
			ConstellationID: constellation.ID,
			Name:            system.Name,
			SecurityStatus:  float64(system.SecurityStatus),
		}
		if err := s.r.CreateEveSolarSystem(ctx, arg); err != nil {
			return nil, err
		}
		return s.r.GetEveSolarSystem(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveSolarSystem), nil
}
