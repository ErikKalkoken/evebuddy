package eveuniverse

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (eu *EveUniverse) GetOrCreateEveRegionESI(ctx context.Context, id int32) (*model.EveRegion, error) {
	x, err := eu.r.GetEveRegion(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveRegionFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverse) createEveRegionFromESI(ctx context.Context, id int32) (*model.EveRegion, error) {
	key := fmt.Sprintf("createEveRegionFromESI-%d", id)
	y, err, _ := eu.singleGroup.Do(key, func() (any, error) {
		region, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseRegionsRegionId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveRegionParams{
			ID:          region.RegionId,
			Description: region.Description,
			Name:        region.Name,
		}
		return eu.r.CreateEveRegion(ctx, arg)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveRegion), nil
}

func (eu *EveUniverse) GetOrCreateEveConstellationESI(ctx context.Context, id int32) (*model.EveConstellation, error) {
	x, err := eu.r.GetEveConstellation(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveConstellationFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverse) createEveConstellationFromESI(ctx context.Context, id int32) (*model.EveConstellation, error) {
	key := fmt.Sprintf("createEveConstellationFromESI-%d", id)
	y, err, _ := eu.singleGroup.Do(key, func() (any, error) {
		constellation, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseConstellationsConstellationId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		_, err = eu.GetOrCreateEveRegionESI(ctx, constellation.RegionId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveConstellationParams{
			ID:       constellation.ConstellationId,
			RegionID: constellation.RegionId,
			Name:     constellation.Name,
		}
		if err := eu.r.CreateEveConstellation(ctx, arg); err != nil {
			return nil, err
		}
		return eu.r.GetEveConstellation(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveConstellation), nil
}

func (eu *EveUniverse) GetOrCreateEveSolarSystemESI(ctx context.Context, id int32) (*model.EveSolarSystem, error) {
	x, err := eu.r.GetEveSolarSystem(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveSolarSystemFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverse) createEveSolarSystemFromESI(ctx context.Context, id int32) (*model.EveSolarSystem, error) {
	key := fmt.Sprintf("createEveSolarSystemFromESI-%d", id)
	y, err, _ := eu.singleGroup.Do(key, func() (any, error) {
		system, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseSystemsSystemId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		constellation, err := eu.GetOrCreateEveConstellationESI(ctx, system.ConstellationId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveSolarSystemParams{
			ID:              system.SystemId,
			ConstellationID: constellation.ID,
			Name:            system.Name,
			SecurityStatus:  float64(system.SecurityStatus),
		}
		if err := eu.r.CreateEveSolarSystem(ctx, arg); err != nil {
			return nil, err
		}
		return eu.r.GetEveSolarSystem(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveSolarSystem), nil
}