package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (s *Service) getOrCreateEveRaceESI(ctx context.Context, id int32) (*model.EveRace, error) {
	x, err := s.r.GetEveRace(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.createEveRaceFromESI(ctx, id)
		}
		return x, err
	}
	return x, nil
}

func (s *Service) createEveRaceFromESI(ctx context.Context, id int32) (*model.EveRace, error) {
	key := fmt.Sprintf("createEveRaceFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		races, _, err := s.esiClient.ESI.UniverseApi.GetUniverseRaces(ctx, nil)
		if err != nil {
			return nil, err
		}
		for _, race := range races {
			if race.RaceId == id {
				return s.r.CreateEveRace(ctx, race.RaceId, race.Description, race.Name)
			}
		}
		return nil, fmt.Errorf("race with ID %d not found", id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveRace), nil
}
