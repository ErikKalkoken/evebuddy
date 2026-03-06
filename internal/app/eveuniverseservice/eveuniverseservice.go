// Package eveuniverseservice contains EVE universe service.
package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/xsingleflight"
)

// EveUniverseService provides access to Eve Online models with on-demand loading from ESI and persistent local caching.
type EveUniverseService struct {
	// Now returns the current time in UTC. Can be overwritten for tests.
	Now func() time.Time

	concurrencyLimit int
	esiClient        *esi.APIClient
	scs              *statuscacheservice.StatusCacheService
	sfg              singleflight.Group
	signals          *app.Signals
	st               *storage.Storage
}

type Params struct {
	ConcurrencyLimit   int // max number of concurrent Goroutines (per group)
	ESIClient          *esi.APIClient
	Signals            *app.Signals
	StatusCacheService *statuscacheservice.StatusCacheService
	Storage            *storage.Storage
}

// New returns a new instance of an Eve universe service.
func New(arg Params) *EveUniverseService {
	if arg.ESIClient == nil {
		panic("ESIClient missing")
	}
	if arg.Signals == nil {
		panic("Signals missing")
	}
	if arg.Storage == nil {
		panic("Storage missing")
	}
	if arg.StatusCacheService == nil {
		panic("StatusCacheService missing")
	}
	s := &EveUniverseService{
		concurrencyLimit: -1, // Default is no limit
		esiClient:        arg.ESIClient,
		Now:              func() time.Time { return time.Now().UTC() },
		scs:              arg.StatusCacheService,
		signals:          arg.Signals,
		st:               arg.Storage,
	}
	if arg.ConcurrencyLimit > 0 {
		s.concurrencyLimit = arg.ConcurrencyLimit
	}
	return s
}
func (s *EveUniverseService) GetOrCreateRaceESI(ctx context.Context, id int64) (*app.EveRace, error) {
	o, err, _ := xsingleflight.Do(&s.sfg, fmt.Sprintf("GetOrCreateRaceESI-%d", id), func() (*app.EveRace, error) {
		o, err := s.st.GetEveRace(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		races, _, err := s.esiClient.UniverseAPI.GetUniverseRaces(ctx).Execute()
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
	return o, nil
}

func (s *EveUniverseService) GetOrCreateSchematicESI(ctx context.Context, id int64) (*app.EveSchematic, error) {
	o, err, _ := xsingleflight.Do(&s.sfg, fmt.Sprintf("GetOrCreateSchematicESI-%d", id), func() (*app.EveSchematic, error) {
		o, err := s.st.GetEveSchematic(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		d, _, err := s.esiClient.PlanetaryInteractionAPI.GetUniverseSchematicsSchematicId(ctx, id).Execute()
		if err != nil {
			return nil, err
		}
		o2, err := s.st.CreateEveSchematic(ctx, storage.CreateEveSchematicParams{
			ID:        id,
			CycleTime: d.CycleTime,
			Name:      d.SchematicName,
		})
		if err != nil {
			return nil, err
		}
		slog.Info("Created eve schematic", "id", id)
		return o2, nil
	})
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *EveUniverseService) AddMissingEveEntitiesAndLocations(ctx context.Context, entityIDs set.Set[int64], locationIDs set.Set[int64]) error {
	g := new(errgroup.Group)
	if entityIDs.Size() > 0 {
		g.Go(func() error {
			_, err := s.AddMissingEntities(ctx, entityIDs)
			return err
		})
	}
	if locationIDs.Size() > 0 {
		g.Go(func() error {
			return s.AddMissingLocations(ctx, locationIDs)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}
