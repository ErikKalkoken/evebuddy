package service

import (
	"context"
	"database/sql"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (s *Service) updateCharacterSkillsESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	skills, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkills(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.hasCharacterSectionChanged(ctx, characterID, model.CharacterSectionSkills, skills)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	totalSP := sql.NullInt64{Int64: skills.TotalSp, Valid: true}
	unallocatedSP := sql.NullInt64{Int64: int64(skills.UnallocatedSp), Valid: true}
	if err := s.r.UpdateCharacterSkillPoints(ctx, characterID, totalSP, unallocatedSP); err != nil {
		return false, err
	}
	var existingSkills []int32
	for _, o := range skills.Skills {
		existingSkills = append(existingSkills, o.SkillId)
		_, err = s.getOrCreateEveTypeESI(ctx, o.SkillId)
		if err != nil {
			return false, err
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
			return false, err
		}
	}
	err = s.r.DeleteExcludedCharacterSkills(ctx, characterID, existingSkills)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) ListCharacterSkillProgress(characterID, eveGroupID int32) ([]model.ListCharacterSkillProgress, error) {
	ctx := context.Background()
	return s.r.ListCharacterSkillProgress(ctx, characterID, eveGroupID)
}

func (s *Service) ListCharacterSkillGroupsProgress(characterID int32) ([]model.ListCharacterSkillGroupProgress, error) {
	ctx := context.Background()
	return s.r.ListCharacterSkillGroupsProgress(ctx, characterID)
}
