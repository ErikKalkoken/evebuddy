package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) GetTotalTrainingTime(ctx context.Context, characterID int32) (optional.Optional[time.Duration], error) {
	return s.st.GetCharacterTotalTrainingTime(ctx, characterID)
}

func (cs *CharacterService) NotifyExpiredTraining(ctx context.Context, characterID int32, notify func(title, content string)) error {
	c, err := cs.GetCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	if !c.IsTrainingWatched {
		return nil
	}
	t, err := cs.GetTotalTrainingTime(ctx, characterID)
	if err != nil {
		return err
	}
	if !t.IsEmpty() {
		return nil
	}
	title := fmt.Sprintf("%s: No skill in training", c.EveCharacter.Name)
	content := "There is currently no skill being trained for this character."
	notify(title, content)
	return cs.UpdateIsTrainingWatched(ctx, characterID, false)
}

func (s *CharacterService) ListSkillqueueItems(ctx context.Context, characterID int32) ([]*app.CharacterSkillqueueItem, error) {
	return s.st.ListCharacterSkillqueueItems(ctx, characterID)
}

// UpdateSkillqueueESI updates the skillqueue for a character from ESI
// and reports wether it has changed.
func (s *CharacterService) UpdateSkillqueueESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
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
			slog.Debug("Received skillqueue from ESI", "characterID", characterID, "items", len(items))
			return items, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			items := data.([]esi.GetCharactersCharacterIdSkillqueue200Ok)
			args := make([]storage.SkillqueueItemParams, len(items))
			for i, o := range items {
				_, err := s.EveUniverseService.GetOrCreateTypeESI(ctx, o.SkillId)
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
			slog.Info("Stored updated skillqueue items", "characterID", characterID, "count", len(args))
			return nil
		})

}
