package service

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (s *Service) ListCharacterImplants(characterID int32) ([]*model.CharacterImplant, error) {
	ctx := context.Background()
	return s.r.ListCharacterImplants(ctx, characterID)
}

func (s *Service) updateCharacterImplantsESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
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
				_, err := s.getOrCreateEveTypeESI(ctx, typeID)
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
