package character

import (
	"context"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) GetCharacterTotalTrainingTime(ctx context.Context, characterID int32) (optional.Optional[time.Duration], error) {
	return s.st.GetCharacterTotalTrainingTime(ctx, characterID)
}

func (s *CharacterService) ListCharacterSkillqueueItems(ctx context.Context, characterID int32) ([]*app.CharacterSkillqueueItem, error) {
	return s.st.ListCharacterSkillqueueItems(ctx, characterID)
}

// UpdateCharacterSkillqueueESI updates the skillqueue for a character from ESI
// and reports wether it has changed.
func (s *CharacterService) UpdateCharacterSkillqueueESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionSkillqueue {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
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
				_, err := s.EveUniverseService.GetOrCreateEveTypeESI(ctx, o.SkillId)
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
			if err := s.st.ReplaceCharacterSkillqueueItems(ctx, characterID, args); err != nil {
				return err
			}
			slog.Info("Updated skillqueue items", "characterID", characterID, "count", len(args))
			return nil
		})

}
