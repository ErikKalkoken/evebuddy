package service

import (
	"context"
	"database/sql"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
)

func (s *Service) updateCharacterSkillsESI(ctx context.Context, characterID int32) (bool, error) {
	return s.updateCharacterSectionIfChanged(
		ctx, characterID, model.CharacterSectionSkills,
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
			if err := s.r.UpdateCharacterSkillPoints(ctx, characterID, totalSP, unallocatedSP); err != nil {
				return err
			}
			var existingSkills []int32
			for _, o := range skills.Skills {
				existingSkills = append(existingSkills, o.SkillId)
				_, err := s.getOrCreateEveTypeESI(ctx, o.SkillId)
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
				err = s.r.UpdateOrCreateCharacterSkill(ctx, arg)
				if err != nil {
					return err
				}
			}
			if err := s.r.DeleteExcludedCharacterSkills(ctx, characterID, existingSkills); err != nil {
				return err
			}
			return nil
		})
}

func (s *Service) ListCharacterSkillProgress(characterID, eveGroupID int32) ([]model.ListCharacterSkillProgress, error) {
	ctx := context.Background()
	return s.r.ListCharacterSkillProgress(ctx, characterID, eveGroupID)
}

func (s *Service) ListCharacterSkillGroupsProgress(characterID int32) ([]model.ListCharacterSkillGroupProgress, error) {
	ctx := context.Background()
	return s.r.ListCharacterSkillGroupsProgress(ctx, characterID)
}
