package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/helper/types"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (s *Service) GetTotalTrainingTime(characterID int32) (types.NullDuration, error) {
	ctx := context.Background()
	return s.r.GetTotalTrainingTime(ctx, characterID)
}

func (s *Service) ListSkillqueueItems(characterID int32) ([]*model.SkillqueueItem, error) {
	ctx := context.Background()
	return s.r.ListSkillqueueItems(ctx, characterID)
}

func (s *Service) UpdateSkillqueueESI(characterID int32) (bool, error) {
	ctx := context.Background()
	key := fmt.Sprintf("UpdateSkillqueueESI-%d", characterID)
	x, err, _ := s.singleGroup.Do(key, func() (any, error) {
		changed, err := s.updateSkillqueue(ctx, characterID)
		if err != nil {
			return changed, fmt.Errorf("failed to update skillqueue from ESI for character %d: %w", characterID, err)
		}
		if err := s.SectionSetUpdated(characterID, model.UpdateSectionSkillqueue); err != nil {
			slog.Warn("Failed to set updated for skillqueue", "err", err)
		}
		return changed, nil
	})
	changed := x.(bool)
	return changed, err
}

func (s *Service) updateSkillqueue(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	items, r, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkillqueue(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	slog.Info("Received skillqueue items from ESI", "count", len(items), "characterID", characterID)
	changed, err := s.hasSectionChanged(ctx, characterID, model.UpdateSectionSkillqueue, r)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	args := make([]storage.SkillqueueItemParams, len(items))
	for i, o := range items {
		_, err := s.getOrCreateEveTypeESI(ctx, o.SkillId)
		if err != nil {
			return false, err
		}
		args[i] = storage.SkillqueueItemParams{
			EveTypeID:       o.SkillId,
			FinishDate:      o.FinishDate,
			FinishedLevel:   int(o.FinishedLevel),
			LevelEndSP:      int(o.LevelEndSp),
			LevelStartSP:    int(o.LevelStartSp),
			MyCharacterID:   characterID,
			QueuePosition:   int(o.QueuePosition),
			StartDate:       o.StartDate,
			TrainingStartSP: int(o.TrainingStartSp),
		}
	}
	if err := s.r.ReplaceSkillqueueItems(ctx, characterID, args); err != nil {
		return false, err
	}
	slog.Info("Updated skillqueue items", "characterID", characterID, "count", len(args))
	return true, nil
}
