package characterservice

import (
	"context"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) GetSkill(ctx context.Context, characterID, typeID int32) (*app.CharacterSkill, error) {
	return s.st.GetCharacterSkill(ctx, characterID, typeID)
}

func (s *CharacterService) ListSkillProgress(ctx context.Context, characterID, eveGroupID int32) ([]app.ListSkillProgress, error) {
	return s.st.ListCharacterSkillProgress(ctx, characterID, eveGroupID)
}

func (s *CharacterService) ListSkillGroupsProgress(ctx context.Context, characterID int32) ([]app.ListCharacterSkillGroupProgress, error) {
	return s.st.ListCharacterSkillGroupsProgress(ctx, characterID)
}

func (s *CharacterService) updateSkillsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionSkills {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			skills, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkills(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			slog.Debug("Received character skills from ESI", "characterID", characterID, "items", len(skills.Skills))
			return skills, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			skills := data.(esi.GetCharactersCharacterIdSkillsOk)
			total := optional.New(int(skills.TotalSp))
			unallocated := optional.New(int(skills.UnallocatedSp))
			if err := s.st.UpdateCharacterSkillPoints(ctx, characterID, total, unallocated); err != nil {
				return err
			}
			currentSkillIDs, err := s.st.ListCharacterSkillIDs(ctx, characterID)
			if err != nil {
				return err
			}
			incomingSkillIDs := set.New[int32]()
			for _, o := range skills.Skills {
				incomingSkillIDs.Add(o.SkillId)
				_, err := s.EveUniverseService.GetOrCreateTypeESI(ctx, o.SkillId)
				if err != nil {
					return err
				}
				arg := storage.UpdateOrCreateCharacterSkillParams{
					CharacterID:        characterID,
					EveTypeID:          o.SkillId,
					ActiveSkillLevel:   int(o.ActiveSkillLevel),
					TrainedSkillLevel:  int(o.TrainedSkillLevel),
					SkillPointsInSkill: int(o.SkillpointsInSkill),
				}
				err = s.st.UpdateOrCreateCharacterSkill(ctx, arg)
				if err != nil {
					return err
				}
			}
			slog.Info("Stored updated character skills", "characterID", characterID, "count", len(skills.Skills))
			if ids := currentSkillIDs.Difference(incomingSkillIDs); ids.Size() > 0 {
				if err := s.st.DeleteCharacterSkills(ctx, characterID, ids.ToSlice()); err != nil {
					return err
				}
				slog.Info("Deleted obsolete character skills", "characterID", characterID, "count", ids.Size())
			}
			return nil
		})
}
