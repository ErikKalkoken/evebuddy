package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
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
				return eu.st.CreateEveRace(ctx, race.RaceId, race.Description, race.Name)
			}
		}
		return nil, fmt.Errorf("race with ID %d not found: %w", id, app.ErrNotFound)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveRace), nil
}
