package character

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) ListCharacterJumpClones(ctx context.Context, characterID int32) ([]*app.CharacterJumpClone, error) {
	return s.st.ListCharacterJumpClones(ctx, characterID)
}

// TODO: Consolidate with updating home in separate function

func (s *CharacterService) updateCharacterJumpClonesESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionJumpClones {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			clones, _, err := s.esiClient.ESI.ClonesApi.GetCharactersCharacterIdClones(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return clones, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			var home optional.Optional[int64]
			clones := data.(esi.GetCharactersCharacterIdClonesOk)
			if clones.HomeLocation.LocationId != 0 {
				_, err := s.EveUniverseService.GetOrCreateEveLocationESI(ctx, clones.HomeLocation.LocationId)
				if err != nil {
					return err
				}
				home.Set(clones.HomeLocation.LocationId)
			}
			if err := s.st.UpdateCharacterHome(ctx, characterID, home); err != nil {
				return err
			}
			args := make([]storage.CreateCharacterJumpCloneParams, len(clones.JumpClones))
			for i, jc := range clones.JumpClones {
				_, err := s.EveUniverseService.GetOrCreateEveLocationESI(ctx, jc.LocationId)
				if err != nil {
					return err
				}
				if err := s.EveUniverseService.AddMissingEveTypes(ctx, jc.Implants); err != nil {
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
