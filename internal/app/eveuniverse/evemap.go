package eveuniverse

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
)

func (eu *EveUniverseService) GetOrCreateEveRegionESI(ctx context.Context, id int32) (*app.EveRegion, error) {
	x, err := eu.st.GetEveRegion(ctx, id)
	if errors.Is(err, sqlite.ErrNotFound) {
		return eu.createEveRegionFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverseService) createEveRegionFromESI(ctx context.Context, id int32) (*app.EveRegion, error) {
	key := fmt.Sprintf("createEveRegionFromESI-%d", id)
	y, err, _ := eu.sfg.Do(key, func() (any, error) {
		region, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseRegionsRegionId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := sqlite.CreateEveRegionParams{
			ID:          region.RegionId,
			Description: region.Description,
			Name:        region.Name,
		}
		return eu.st.CreateEveRegion(ctx, arg)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveRegion), nil
}

func (eu *EveUniverseService) GetOrCreateEveConstellationESI(ctx context.Context, id int32) (*app.EveConstellation, error) {
	x, err := eu.st.GetEveConstellation(ctx, id)
	if errors.Is(err, sqlite.ErrNotFound) {
		return eu.createEveConstellationFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverseService) createEveConstellationFromESI(ctx context.Context, id int32) (*app.EveConstellation, error) {
	key := fmt.Sprintf("createEveConstellationFromESI-%d", id)
	y, err, _ := eu.sfg.Do(key, func() (any, error) {
		constellation, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseConstellationsConstellationId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		_, err = eu.GetOrCreateEveRegionESI(ctx, constellation.RegionId)
		if err != nil {
			return nil, err
		}
		arg := sqlite.CreateEveConstellationParams{
			ID:       constellation.ConstellationId,
			RegionID: constellation.RegionId,
			Name:     constellation.Name,
		}
		if err := eu.st.CreateEveConstellation(ctx, arg); err != nil {
			return nil, err
		}
		return eu.st.GetEveConstellation(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveConstellation), nil
}

func (eu *EveUniverseService) GetOrCreateEveSolarSystemESI(ctx context.Context, id int32) (*app.EveSolarSystem, error) {
	x, err := eu.st.GetEveSolarSystem(ctx, id)
	if errors.Is(err, sqlite.ErrNotFound) {
		return eu.createEveSolarSystemFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverseService) createEveSolarSystemFromESI(ctx context.Context, id int32) (*app.EveSolarSystem, error) {
	key := fmt.Sprintf("createEveSolarSystemFromESI-%d", id)
	y, err, _ := eu.sfg.Do(key, func() (any, error) {
		system, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseSystemsSystemId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		constellation, err := eu.GetOrCreateEveConstellationESI(ctx, system.ConstellationId)
		if err != nil {
			return nil, err
		}
		arg := sqlite.CreateEveSolarSystemParams{
			ID:              system.SystemId,
			ConstellationID: constellation.ID,
			Name:            system.Name,
			SecurityStatus:  system.SecurityStatus,
		}
		if err := eu.st.CreateEveSolarSystem(ctx, arg); err != nil {
			return nil, err
		}
		return eu.st.GetEveSolarSystem(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveSolarSystem), nil
}
