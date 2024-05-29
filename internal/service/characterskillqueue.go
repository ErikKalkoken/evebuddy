package service

import (
	"context"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/helper/types"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
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
	return s.updateCharacterSectionIfChanged(
		ctx, characterID, model.CharacterSectionSkillqueue,
		func(ctx context.Context, characterID int32) (any, error) {
			items, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkillqueue(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			slog.Info("Received skillqueue from ESI", "items", len(items), "characterID", characterID)
			return items, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			items := data.([]esi.GetCharactersCharacterIdSkillqueue200Ok)
			args := make([]storage.SkillqueueItemParams, len(items))
			for i, o := range items {
				_, err := s.getOrCreateEveTypeESI(ctx, o.SkillId)
				if err != nil {
					return err
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
				return err
			}
			slog.Info("Updated skillqueue items", "characterID", characterID, "count", len(args))
			return nil
		})

}
