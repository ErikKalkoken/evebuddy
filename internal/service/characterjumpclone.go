package service

import (
	"context"
	"database/sql"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (s *Service) ListCharacterJumpClones(characterID int32) ([]*model.CharacterJumpClone, error) {
	ctx := context.Background()
	return s.r.ListCharacterJumpClones(ctx, characterID)
}

// TODO: Consolidate with updating home in separate function

func (s *Service) updateCharacterJumpClonesESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	clones, _, err := s.esiClient.ESI.ClonesApi.GetCharactersCharacterIdClones(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.recordCharacterSectionUpdate(ctx, characterID, model.CharacterSectionJumpClones, clones)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	var home sql.NullInt64
	if clones.HomeLocation.LocationId != 0 {
		_, err = s.getOrCreateLocationESI(ctx, clones.HomeLocation.LocationId)
		if err != nil {
			return false, err
		}
		home.Int64 = clones.HomeLocation.LocationId
		home.Valid = true
	}
	if err := s.r.UpdateCharacterHome(ctx, characterID, home); err != nil {
		return false, err
	}
	args := make([]storage.CreateCharacterJumpCloneParams, len(clones.JumpClones))
	for i, jc := range clones.JumpClones {
		_, err = s.getOrCreateLocationESI(ctx, jc.LocationId)
		if err != nil {
			return false, err
		}
		for _, typeID := range jc.Implants {
			_, err = s.getOrCreateEveTypeESI(ctx, typeID)
			if err != nil {
				return false, err
			}
		}
		args[i] = storage.CreateCharacterJumpCloneParams{
			CharacterID: characterID,
			LocationID:  jc.LocationId,
			JumpCloneID: int64(jc.JumpCloneId),
			Implants:    jc.Implants,
		}
	}
	if err = s.r.ReplaceCharacterJumpClones(ctx, characterID, args); err != nil {
		return false, err
	}
	return true, nil
}
