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

func (s *Service) updateCharacterImplantsESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	implants, _, err := s.esiClient.ESI.ClonesApi.GetCharactersCharacterIdImplants(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.recordCharacterSectionUpdate(ctx, characterID, model.CharacterSectionImplants, implants)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	args := make([]storage.CreateCharacterImplantParams, len(implants))
	for i, typeID := range implants {
		_, err = s.getOrCreateEveTypeESI(ctx, typeID)
		if err != nil {
			return false, err
		}
		args[i] = storage.CreateCharacterImplantParams{
			CharacterID: characterID,
			EveTypeID:   typeID,
		}
	}
	err = s.r.ReplaceCharacterImplants(ctx, characterID, args)
	if err != nil {
		return false, err
	}
	return true, nil
}
