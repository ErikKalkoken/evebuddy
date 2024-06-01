package service

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
)

func (s *Service) GetCharacterAttributes(characterID int32) (*model.CharacterAttributes, error) {
	ctx := context.Background()
	return s.r.GetCharacterAttributes(ctx, characterID)
}

func (s *Service) updateCharacterAttributesESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
	return s.updateCharacterSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			attributes, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdAttributes(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return attributes, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			attributes := data.(esi.GetCharactersCharacterIdAttributesOk)
			arg := storage.UpdateOrCreateCharacterAttributesParams{
				CharacterID:   characterID,
				BonusRemaps:   int(attributes.BonusRemaps),
				Charisma:      int(attributes.Charisma),
				Intelligence:  int(attributes.Intelligence),
				LastRemapDate: attributes.LastRemapDate,
				Memory:        int(attributes.Memory),
				Perception:    int(attributes.Perception),
				Willpower:     int(attributes.Willpower),
			}
			if err := s.r.UpdateOrCreateCharacterAttributes(ctx, arg); err != nil {
				return err
			}
			return nil
		})
}
