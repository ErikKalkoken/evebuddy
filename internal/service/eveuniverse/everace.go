package eveuniverse

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"
)

var ErrNotFound = errors.New("object not found")

type EveUniverse struct {
	esiClient   *goesi.APIClient
	singleGroup *singleflight.Group
	r           *storage.Storage
}

func New(r *storage.Storage, esiClient *goesi.APIClient) *EveUniverse {
	eu := &EveUniverse{
		esiClient:   esiClient,
		r:           r,
		singleGroup: new(singleflight.Group),
	}
	return eu
}

func (eu *EveUniverse) GetOrCreateEveRaceESI(ctx context.Context, id int32) (*model.EveRace, error) {
	x, err := eu.r.GetEveRace(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveRaceFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverse) createEveRaceFromESI(ctx context.Context, id int32) (*model.EveRace, error) {
	key := fmt.Sprintf("createEveRaceFromESI-%d", id)
	y, err, _ := eu.singleGroup.Do(key, func() (any, error) {
		races, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseRaces(ctx, nil)
		if err != nil {
			return nil, err
		}
		for _, race := range races {
			if race.RaceId == id {
				return eu.r.CreateEveRace(ctx, race.RaceId, race.Description, race.Name)
			}
		}
		return nil, fmt.Errorf("race with ID %d not found: %w", id, ErrNotFound)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveRace), nil
}