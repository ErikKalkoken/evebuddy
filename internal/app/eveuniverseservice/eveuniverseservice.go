// Package eveuniverseservice contains EVE universe service.
package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"time"

	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

// EveUniverseService provides access to Eve Online models with on-demand loading from ESI and persistent local caching.
type EveUniverseService struct {
	// Now returns the current time in UTC. Can be overwritten for tests.
	Now func() time.Time

	esiClient *goesi.APIClient
	scs       *statuscacheservice.StatusCacheService
	sfg       *singleflight.Group
	st        *storage.Storage
}

type Params struct {
	ESIClient          *goesi.APIClient
	StatusCacheService *statuscacheservice.StatusCacheService
	Storage            *storage.Storage
}

// New returns a new instance of an Eve universe service.
func New(args Params) *EveUniverseService {
	eu := &EveUniverseService{
		scs:       args.StatusCacheService,
		esiClient: args.ESIClient,
		st:        args.Storage,
		sfg:       new(singleflight.Group),
		Now: func() time.Time {
			return time.Now().UTC()
		},
	}
	return eu
}
func (s *EveUniverseService) GetOrCreateRaceESI(ctx context.Context, id int32) (*app.EveRace, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateRaceESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveRace(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		races, _, err := s.esiClient.ESI.UniverseApi.GetUniverseRaces(ctx, nil)
		if err != nil {
			return nil, err
		}
		for _, race := range races {
			if race.RaceId == id {
				arg := storage.CreateEveRaceParams{
					ID:          race.RaceId,
					Description: race.Description,
					Name:        race.Name,
				}
				o, err := s.st.CreateEveRace(ctx, arg)
				if err != nil {
					return nil, err
				}
				slog.Info("Created eve race", "id", id)
				return o, nil
			}
		}
		return nil, fmt.Errorf("race with ID %d not found: %w", id, app.ErrNotFound)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveRace), nil
}

func (s *EveUniverseService) GetOrCreateSchematicESI(ctx context.Context, id int32) (*app.EveSchematic, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateSchematicESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveSchematic(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		d, _, err := s.esiClient.ESI.PlanetaryInteractionApi.GetUniverseSchematicsSchematicId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveSchematicParams{
			ID:        id,
			CycleTime: int(d.CycleTime),
			Name:      d.SchematicName,
		}
		o2, err := s.st.CreateEveSchematic(ctx, arg)
		if err != nil {
			return nil, err
		}
		slog.Info("Created eve schematic", "id", id)
		return o2, nil
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveSchematic), nil
}
