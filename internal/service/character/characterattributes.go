package character

import (
	"context"
	"errors"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) GetCharacterAttributes(ctx context.Context, characterID int32) (*model.CharacterAttributes, error) {
	o, err := s.st.GetCharacterAttributes(ctx, characterID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	return o, err
}

func (s *CharacterService) updateCharacterAttributesESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
	if arg.Section != model.SectionAttributes {
		panic("called with wrong section")
	}
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
			if err := s.st.UpdateOrCreateCharacterAttributes(ctx, arg); err != nil {
				return err
			}
			return nil
		})
}
