package service

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (s *Service) GetCharacterAttributes(characterID int32) (*model.CharacterAttributes, error) {
	ctx := context.Background()
	return s.r.GetCharacterAttributes(ctx, characterID)
}

func (s *Service) updateCharacterAttributesESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	attributes, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdAttributes(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.recordCharacterSectionUpdate(ctx, characterID, model.CharacterSectionAttributes, attributes)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
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
	err = s.r.UpdateOrCreateCharacterAttributes(ctx, arg)
	if err != nil {
		return false, err
	}
	return true, nil
}
