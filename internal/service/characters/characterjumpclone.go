package characters

import (
	"context"
	"database/sql"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
)

func (s *Characters) ListCharacterJumpClones(ctx context.Context, characterID int32) ([]*model.CharacterJumpClone, error) {
	return s.st.ListCharacterJumpClones(ctx, characterID)
}

// TODO: Consolidate with updating home in separate function

func (s *Characters) updateCharacterJumpClonesESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
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
				_, err := s.eu.GetOrCreateLocationESI(ctx, clones.HomeLocation.LocationId)
				if err != nil {
					return err
				}
				home.Int64 = clones.HomeLocation.LocationId
				home.Valid = true
			}
			if err := s.st.UpdateCharacterHome(ctx, characterID, home); err != nil {
				return err
			}
			args := make([]storage.CreateCharacterJumpCloneParams, len(clones.JumpClones))
			for i, jc := range clones.JumpClones {
				_, err := s.eu.GetOrCreateLocationESI(ctx, jc.LocationId)
				if err != nil {
					return err
				}
				if err := s.eu.AddMissingEveTypes(ctx, jc.Implants); err != nil {
					return err
				}
				args[i] = storage.CreateCharacterJumpCloneParams{
					CharacterID: characterID,
					LocationID:  jc.LocationId,
					JumpCloneID: int64(jc.JumpCloneId),
					Implants:    jc.Implants,
				}
			}
			if err := s.st.ReplaceCharacterJumpClones(ctx, characterID, args); err != nil {
				return err
			}
			return nil
		})
}
