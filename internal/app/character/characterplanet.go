package character

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) ListCharacterPlanets(ctx context.Context, characterID int32) ([]*app.CharacterPlanet, error) {
	return s.st.ListCharacterPlanets(ctx, characterID)
}

// TODO: Improve update logic to only update changes

func (s *CharacterService) updateCharacterPlanetsESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionPlanets {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			skills, _, err := s.esiClient.ESI.PlanetaryInteractionApi.GetCharactersCharacterIdPlanets(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return skills, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			planets := data.([]esi.GetCharactersCharacterIdPlanets200Ok)
			args := make([]storage.CreateCharacterPlanetParams, len(planets))
			for i, o := range planets {
				_, err := s.EveUniverseService.GetOrCreateEvePlanetESI(ctx, o.PlanetId)
				if err != nil {
					return err
				}
				args[i] = storage.CreateCharacterPlanetParams{
					CharacterID:  characterID,
					EvePlanetID:  o.PlanetId,
					LastUpdate:   o.LastUpdate,
					NumPins:      int(o.NumPins),
					UpgradeLevel: int(o.UpgradeLevel),
				}
			}
			if err := s.st.ReplaceCharacterPlanets(ctx, characterID, args); err != nil {
				return err
			}
			return nil
		})
}
