package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

func (eu *EveUniverseService) GetOrCreateRaceESI(ctx context.Context, id int32) (*app.EveRace, error) {
	o, err := eu.st.GetEveRace(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return eu.createEveRaceFromESI(ctx, id)
	}
	return o, err
}

func (eu *EveUniverseService) createEveRaceFromESI(ctx context.Context, id int32) (*app.EveRace, error) {
	key := fmt.Sprintf("createEveRaceFromESI-%d", id)
	y, err, _ := eu.sfg.Do(key, func() (any, error) {
		races, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseRaces(ctx, nil)
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
				return eu.st.CreateEveRace(ctx, arg)
			}
		}
		return nil, fmt.Errorf("race with ID %d not found: %w", id, app.ErrNotFound)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveRace), nil
}
