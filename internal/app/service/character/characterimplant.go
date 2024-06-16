package character

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

func (s *CharacterService) ListCharacterImplants(ctx context.Context, characterID int32) ([]*app.CharacterImplant, error) {
	return s.st.ListCharacterImplants(ctx, characterID)
}

func (s *CharacterService) updateCharacterImplantsESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionImplants {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			implants, _, err := s.esiClient.ESI.ClonesApi.GetCharactersCharacterIdImplants(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return implants, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			implants := data.([]int32)
			args := make([]storage.CreateCharacterImplantParams, len(implants))
			for i, typeID := range implants {
				_, err := s.eu.GetOrCreateEveTypeESI(ctx, typeID)
				if err != nil {
					return err
				}
				args[i] = storage.CreateCharacterImplantParams{
					CharacterID: characterID,
					EveTypeID:   typeID,
				}
			}
			if err := s.st.ReplaceCharacterImplants(ctx, characterID, args); err != nil {
				return err
			}
			return nil
		})
}
