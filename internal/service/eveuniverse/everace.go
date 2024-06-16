package eveuniverse

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (eu *EveUniverseService) GetOrCreateEveRaceESI(ctx context.Context, id int32) (*app.EveRace, error) {
	x, err := eu.st.GetEveRace(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveRaceFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
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
				return eu.st.CreateEveRace(ctx, race.RaceId, race.Description, race.Name)
			}
		}
		return nil, fmt.Errorf("race with ID %d not found: %w", id, ErrNotFound)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveRace), nil
}
