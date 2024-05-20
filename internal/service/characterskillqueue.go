package service

import (
	"context"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/helper/types"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (s *Service) GetCharacterTotalTrainingTime(characterID int32) (types.NullDuration, error) {
	ctx := context.Background()
	return s.r.GetTotalTrainingTime(ctx, characterID)
}

func (s *Service) ListCharacterSkillqueueItems(characterID int32) ([]*model.CharacterSkillqueueItem, error) {
	ctx := context.Background()
	return s.r.ListSkillqueueItems(ctx, characterID)
}

// updateCharacterSkillqueueESI updates the skillqueue for a character from ESI
// and reports wether it has changed.
func (s *Service) updateCharacterSkillqueueESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	items, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkillqueue(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	slog.Info("Received skillqueue from ESI", "items", len(items), "characterID", characterID)
	changed, err := s.hasCharacterSectionChanged(ctx, characterID, model.CharacterSectionSkillqueue, items)
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
			CharacterID:     characterID,
			QueuePosition:   int(o.QueuePosition),
			StartDate:       o.StartDate,
			TrainingStartSP: int(o.TrainingStartSp),
		}
	}
	if err := s.r.ReplaceCharacterSkillqueueItems(ctx, characterID, args); err != nil {
		return false, err
	}
	slog.Info("Updated skillqueue items", "characterID", characterID, "count", len(args))
	return true, nil
}
