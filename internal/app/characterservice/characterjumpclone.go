package characterservice

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) GetCharacterJumpClone(ctx context.Context, characterID, cloneID int32) (*app.CharacterJumpClone, error) {
	return s.st.GetCharacterJumpClone(ctx, characterID, cloneID)
}

func (s *CharacterService) ListAllCharacterJumpClones(ctx context.Context) ([]*app.CharacterJumpClone2, error) {
	return s.st.ListAllCharacterJumpClones(ctx)
}

func (s *CharacterService) ListCharacterJumpClones(ctx context.Context, characterID int32) ([]*app.CharacterJumpClone, error) {
	return s.st.ListCharacterJumpClones(ctx, characterID)
}

// calcCharacterNextCloneJump returns when the next clone jump is available.
// It returns a zero time when a jump is available now.
// It returns empty when a jump could not be calculated.
func (s *CharacterService) calcCharacterNextCloneJump(ctx context.Context, c *app.Character) (optional.Optional[time.Time], error) {
	var z optional.Optional[time.Time]

	if c.LastCloneJumpAt.IsEmpty() {
		return z, nil
	}
	lastJump := c.LastCloneJumpAt.MustValue()

	var skillLevel int
	sk, err := s.GetCharacterSkill(ctx, c.ID, app.EveTypeInfomorphSynchronizing)
	if errors.Is(err, app.ErrNotFound) {
		skillLevel = 0
	} else if err != nil {
		return z, err
	} else {
		skillLevel = sk.ActiveSkillLevel
	}

	nextJump := lastJump.Add(time.Duration(24-skillLevel) * time.Hour)
	if nextJump.Before(time.Now()) {
		return optional.New(time.Time{}), nil
	}
	return optional.New(nextJump), nil
}

// TODO: Consolidate with updating home in separate function

func (s *CharacterService) updateCharacterJumpClonesESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
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
			slog.Debug("Received jump clones from ESI", "characterID", characterID, "count", len(clones.JumpClones))
			return clones, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			clones := data.(esi.GetCharactersCharacterIdClonesOk)
			var home optional.Optional[int64]
			if clones.HomeLocation.LocationId != 0 {
				_, err := s.EveUniverseService.GetOrCreateLocationESI(ctx, clones.HomeLocation.LocationId)
				if err != nil {
					return err
				}
				home.Set(clones.HomeLocation.LocationId)
			}
			if err := s.st.UpdateCharacterHome(ctx, characterID, home); err != nil {
				return err
			}
			if err := s.st.UpdateCharacterLastCloneJump(ctx, characterID, optional.New(clones.LastCloneJumpDate)); err != nil {
				return err
			}
			args := make([]storage.CreateCharacterJumpCloneParams, len(clones.JumpClones))
			for i, jc := range clones.JumpClones {
				_, err := s.EveUniverseService.GetOrCreateLocationESI(ctx, jc.LocationId)
				if err != nil {
					return err
				}
				if err := s.EveUniverseService.AddMissingTypes(ctx, jc.Implants); err != nil {
					return err
				}
				args[i] = storage.CreateCharacterJumpCloneParams{
					CharacterID: characterID,
					Implants:    jc.Implants,
					JumpCloneID: int64(jc.JumpCloneId),
					LocationID:  jc.LocationId,
					Name:        jc.Name,
				}
			}
			if err := s.st.ReplaceCharacterJumpClones(ctx, characterID, args); err != nil {
				return err
			}
			slog.Info("Stored updated jump clones", "characterID", characterID, "count", len(clones.JumpClones))
			return nil
		})
}
