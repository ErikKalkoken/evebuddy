package character

import (
	"context"
	"database/sql"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) updateCharacterSkillsESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
	if arg.Section != model.CharacterSectionSkills {
		panic("called with wrong section")
	}
	return s.updateCharacterSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			skills, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkills(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return skills, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			skills := data.(esi.GetCharactersCharacterIdSkillsOk)
			totalSP := sql.NullInt64{Int64: skills.TotalSp, Valid: true}
			unallocatedSP := sql.NullInt64{Int64: int64(skills.UnallocatedSp), Valid: true}
			if err := s.st.UpdateCharacterSkillPoints(ctx, characterID, totalSP, unallocatedSP); err != nil {
				return err
			}
			var existingSkills []int32
			for _, o := range skills.Skills {
				existingSkills = append(existingSkills, o.SkillId)
				_, err := s.eu.GetOrCreateEveTypeESI(ctx, o.SkillId)
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
			if err := s.st.DeleteExcludedCharacterSkills(ctx, characterID, existingSkills); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) ListCharacterSkillProgress(ctx context.Context, characterID, eveGroupID int32) ([]model.ListCharacterSkillProgress, error) {
	return s.st.ListCharacterSkillProgress(ctx, characterID, eveGroupID)
}

func (s *CharacterService) ListCharacterSkillGroupsProgress(ctx context.Context, characterID int32) ([]model.ListCharacterSkillGroupProgress, error) {
	return s.st.ListCharacterSkillGroupsProgress(ctx, characterID)
}
