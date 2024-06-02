package service

import (
	"context"
	"database/sql"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
)

func (s *Service) ListCharacterJumpClones(ctx context.Context, characterID int32) ([]*model.CharacterJumpClone, error) {
	return s.r.ListCharacterJumpClones(ctx, characterID)
}

// TODO: Consolidate with updating home in separate function

func (s *Service) updateCharacterJumpClonesESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
	if arg.Section != model.CharacterSectionJumpClones {
		panic("called with wrong section")
	}
	return s.updateCharacterSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			clones, _, err := s.esiClient.ESI.ClonesApi.GetCharactersCharacterIdClones(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return clones, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			var home sql.NullInt64
			clones := data.(esi.GetCharactersCharacterIdClonesOk)
			if clones.HomeLocation.LocationId != 0 {
				_, err := s.EveUniverse.GetOrCreateLocationESI(ctx, clones.HomeLocation.LocationId)
				if err != nil {
					return err
				}
				home.Int64 = clones.HomeLocation.LocationId
				home.Valid = true
			}
			if err := s.r.UpdateCharacterHome(ctx, characterID, home); err != nil {
				return err
			}
			args := make([]storage.CreateCharacterJumpCloneParams, len(clones.JumpClones))
			for i, jc := range clones.JumpClones {
				_, err := s.EveUniverse.GetOrCreateLocationESI(ctx, jc.LocationId)
				if err != nil {
					return err
				}
				if err := s.EveUniverse.AddMissingEveTypes(ctx, jc.Implants); err != nil {
					return err
				}
				args[i] = storage.CreateCharacterJumpCloneParams{
					CharacterID: characterID,
					LocationID:  jc.LocationId,
					JumpCloneID: int64(jc.JumpCloneId),
					Implants:    jc.Implants,
				}
			}
			if err := s.r.ReplaceCharacterJumpClones(ctx, characterID, args); err != nil {
				return err
			}
			return nil
		})
}
