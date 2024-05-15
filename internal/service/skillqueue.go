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

func (s *Service) UpdateSkillqueueESI(characterID int32) error {
	ctx := context.Background()
	key := fmt.Sprintf("UpdateSkillqueueESI-%d", characterID)
	_, err, _ := s.singleGroup.Do(key, func() (any, error) {
		x, err := s.updateSkillqueue(ctx, characterID)
		if err != nil {
			return x, fmt.Errorf("failed to update skillqueue from ESI for character %d: %w", characterID, err)
		}
		if err := s.SectionSetUpdated(characterID, model.UpdateSectionSkillqueue); err != nil {
			slog.Warn("Failed to set updated for skillqueue", "err", err)
		}
		return x, nil
	})
	return err
}

func (s *Service) updateSkillqueue(ctx context.Context, characterID int32) (int, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return 0, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	items, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkillqueue(ctx, characterID, nil)
	if err != nil {
		return 0, err
	}
	slog.Info("Received skillqueue items from ESI", "count", len(items), "characterID", characterID)
	args := make([]storage.SkillqueueItemParams, len(items))
	for i, o := range items {
		_, err := s.getOrCreateEveTypeESI(ctx, o.SkillId)
		if err != nil {
			return 0, err
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
		return 0, err
	}
	slog.Info("Updated skillqueue items", "characterID", characterID, "count", len(args))
	return len(args), nil
}
