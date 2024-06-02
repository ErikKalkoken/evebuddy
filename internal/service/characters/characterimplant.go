package characters

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (s *Characters) ListCharacterImplants(ctx context.Context, characterID int32) ([]*model.CharacterImplant, error) {
	return s.r.ListCharacterImplants(ctx, characterID)
}

func (s *Characters) updateCharacterImplantsESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
	if arg.Section != model.CharacterSectionImplants {
		panic("called with wrong section")
	}
	return s.updateCharacterSectionIfChanged(
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
				_, err := s.EveUniverse.GetOrCreateEveTypeESI(ctx, typeID)
				if err != nil {
					return err
				}
				args[i] = storage.CreateCharacterImplantParams{
					CharacterID: characterID,
					EveTypeID:   typeID,
				}
			}
			if err := s.r.ReplaceCharacterImplants(ctx, characterID, args); err != nil {
				return err
			}
			return nil
		})
}
