package service

import (
	"context"
	"errors"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
	"fmt"
)

func (s *Service) GetOrCreateEveRegionESI(id int32) (model.EveRegion, error) {
	ctx := context.Background()
	return s.getOrCreateEveRegionESI(ctx, id)
}

func (s *Service) getOrCreateEveRegionESI(ctx context.Context, id int32) (model.EveRegion, error) {
	x, err := s.r.GetEveRegion(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.createEveRegionFromESI(ctx, id)
		}
		return x, err
	}
	return x, nil
}

func (s *Service) createEveRegionFromESI(ctx context.Context, id int32) (model.EveRegion, error) {
	var dummy model.EveRegion
	key := fmt.Sprintf("createEveRegionFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		r, _, err := s.esiClient.ESI.UniverseApi.GetUniverseRegionsRegionId(ctx, id, nil)
		if err != nil {
			return dummy, err
		}
		return s.r.CreateEveRegion(ctx, r.Description, r.RegionId, r.Name)
	})
	if err != nil {
		return dummy, err
	}
	return y.(model.EveRegion), nil
}

func (s *Service) GetOrCreateEveConstellationESI(id int32) (model.EveConstellation, error) {
	ctx := context.Background()
	return s.getOrCreateEveConstellationESI(ctx, id)
}

func (s *Service) getOrCreateEveConstellationESI(ctx context.Context, id int32) (model.EveConstellation, error) {
	x, err := s.r.GetEveConstellation(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.createEveConstellationFromESI(ctx, id)
		}
		return x, err
	}
	return x, nil
}

func (s *Service) createEveConstellationFromESI(ctx context.Context, id int32) (model.EveConstellation, error) {
	var dummy model.EveConstellation
	key := fmt.Sprintf("createEveConstellationFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		r, _, err := s.esiClient.ESI.UniverseApi.GetUniverseConstellationsConstellationId(ctx, id, nil)
		if err != nil {
			return dummy, err
		}
		c, err := s.getOrCreateEveRegionESI(ctx, r.RegionId)
		if err != nil {
			return dummy, err
		}
		if err := s.r.CreateEveConstellation(ctx, r.ConstellationId, c.ID, r.Name); err != nil {
			return dummy, err
		}
		return s.r.GetEveConstellation(ctx, id)
	})
	if err != nil {
		return dummy, err
	}
	return y.(model.EveConstellation), nil
}

func (s *Service) GetOrCreateEveSolarSystemESI(id int32) (model.EveSolarSystem, error) {
	ctx := context.Background()
	return s.getOrCreateEveSolarSystemESI(ctx, id)
}

func (s *Service) getOrCreateEveSolarSystemESI(ctx context.Context, id int32) (model.EveSolarSystem, error) {
	x, err := s.r.GetEveSolarSystem(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.createEveSolarSystemFromESI(ctx, id)
		}
		return x, err
	}
	return x, nil
}

func (s *Service) createEveSolarSystemFromESI(ctx context.Context, id int32) (model.EveSolarSystem, error) {
	var dummy model.EveSolarSystem
	key := fmt.Sprintf("createEveSolarSystemFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		r, _, err := s.esiClient.ESI.UniverseApi.GetUniverseSystemsSystemId(ctx, id, nil)
		if err != nil {
			return dummy, err
		}
		g, err := s.getOrCreateEveConstellationESI(ctx, r.ConstellationId)
		if err != nil {
			return dummy, err
		}
		if err := s.r.CreateEveSolarSystem(ctx, r.SystemId, g.ID, r.Name, float64(r.SecurityStatus)); err != nil {
			return dummy, err
		}
		return s.r.GetEveSolarSystem(ctx, id)
	})
	if err != nil {
		return dummy, err
	}
	return y.(model.EveSolarSystem), nil
}
