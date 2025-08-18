package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) GetAttributes(ctx context.Context, characterID int32) (*app.CharacterAttributes, error) {
	return s.st.GetCharacterAttributes(ctx, characterID)
}

func (s *CharacterService) updateAttributesESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterAttributes {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			attributes, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdAttributes(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return attributes, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			attributes := data.(esi.GetCharactersCharacterIdAttributesOk)
			arg := storage.UpdateOrCreateCharacterAttributesParams{
				CharacterID:   characterID,
				BonusRemaps:   int(attributes.BonusRemaps),
				Charisma:      int(attributes.Charisma),
				Intelligence:  int(attributes.Intelligence),
				LastRemapDate: attributes.LastRemapDate,
				Memory:        int(attributes.Memory),
				Perception:    int(attributes.Perception),
				Willpower:     int(attributes.Willpower),
			}
			if err := s.st.UpdateOrCreateCharacterAttributes(ctx, arg); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) ListShipsAbilities(ctx context.Context, characterID int32, search string) ([]*app.CharacterShipAbility, error) {
	return s.st.ListCharacterShipsAbilities(ctx, characterID, search)
}

func (s *CharacterService) GetSkill(ctx context.Context, characterID, typeID int32) (*app.CharacterSkill, error) {
	return s.st.GetCharacterSkill(ctx, characterID, typeID)
}

func (s *CharacterService) ListAllCharactersIndustrySlots(ctx context.Context, typ app.IndustryJobType) ([]app.CharacterIndustrySlots, error) {
	total := make(map[int32]int)
	switch typ {
	case app.ManufacturingJob:
		industry1, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeIndustry)
		if err != nil {
			return nil, err
		}
		for _, r := range industry1 {
			if r.Level > 0 {
				total[r.CharacterID] += 1
			}
		}
		industry2, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeMassProduction)
		if err != nil {
			return nil, err
		}
		for _, r := range industry2 {
			total[r.CharacterID] += r.Level
		}
		industry3, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeAdvancedMassProduction)
		if err != nil {
			return nil, err
		}
		for _, r := range industry3 {
			total[r.CharacterID] += r.Level
		}
	case app.ScienceJob:
		research1, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeLaboratoryOperation)
		if err != nil {
			return nil, err
		}
		for _, r := range research1 {
			total[r.CharacterID] += r.Level + 1 // also adds base slot
		}
		research2, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeAdvancedLaboratoryOperation)
		if err != nil {
			return nil, err
		}
		for _, r := range research2 {
			total[r.CharacterID] += r.Level
		}
	case app.ReactionJob:
		reactions1, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeMassReactions)
		if err != nil {
			return nil, err
		}
		for _, r := range reactions1 {
			total[r.CharacterID] += r.Level + 1 // also adds base slot
		}
		reactions2, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeAdvancedMassReactions)
		if err != nil {
			return nil, err
		}
		for _, r := range reactions2 {
			total[r.CharacterID] += r.Level
		}
	}
	characters, err := s.st.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	results := make(map[int32]app.CharacterIndustrySlots)
	for _, c := range characters {
		results[c.ID] = app.CharacterIndustrySlots{
			CharacterID:   c.ID,
			CharacterName: c.Name,
			Type:          typ,
			Total:         total[c.ID],
		}
	}
	counts, err := s.st.ListAllCharacterIndustryJobActiveCounts(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range counts {
		if !typ.Activities().Contains(r.Activity) {
			continue
		}
		x := results[r.InstallerID]
		switch r.Status {
		case app.JobActive:
			x.Busy += r.Count
		case app.JobReady:
			x.Ready += r.Count
		}
		results[r.InstallerID] = x
	}
	for id, r := range results {
		r.Free = r.Total - r.Busy - r.Ready
		results[id] = r
	}
	rows := slices.SortedFunc(maps.Values(results), func(a, b app.CharacterIndustrySlots) int {
		return strings.Compare(a.CharacterName, b.CharacterName)
	})
	return rows, nil
}

func (s *CharacterService) ListSkillProgress(ctx context.Context, characterID, eveGroupID int32) ([]app.ListSkillProgress, error) {
	return s.st.ListCharacterSkillProgress(ctx, characterID, eveGroupID)
}

func (s *CharacterService) ListSkillGroupsProgress(ctx context.Context, characterID int32) ([]app.ListCharacterSkillGroupProgress, error) {
	return s.st.ListCharacterSkillGroupsProgress(ctx, characterID)
}

func (s *CharacterService) updateSkillsESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterSkills {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
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
			incomingSkillIDs := set.Of[int32]()
			for _, o := range skills.Skills {
				incomingSkillIDs.Add(o.SkillId)
				_, err := s.eus.GetOrCreateTypeESI(ctx, o.SkillId)
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
			if ids := set.Difference(currentSkillIDs, incomingSkillIDs); ids.Size() > 0 {
				if err := s.st.DeleteCharacterSkills(ctx, characterID, ids.Slice()); err != nil {
					return err
				}
				slog.Info("Deleted obsolete character skills", "characterID", characterID, "count", ids.Size())
			}
			return nil
		})
}

// IsTrainingActive reports whether training is active.
func (s *CharacterService) IsTrainingActive(ctx context.Context, characterID int32) (bool, error) {
	queue := app.NewCharacterSkillqueue()
	if err := queue.Update(ctx, s, characterID); err != nil {
		return false, err
	}
	return queue.IsActive(), nil
}

func (s *CharacterService) NotifyExpiredTraining(ctx context.Context, characterID int32, notify func(title, content string)) error {
	_, err, _ := s.sfg.Do(fmt.Sprintf("NotifyExpiredTraining-%d", characterID), func() (any, error) {
		c, err := s.GetCharacter(ctx, characterID)
		if err != nil {
			return nil, err
		}
		if !c.IsTrainingWatched {
			return nil, nil
		}
		isActive, err := s.IsTrainingActive(ctx, characterID)
		if err != nil {
			return nil, err
		}
		if isActive {
			return nil, nil
		}
		title := fmt.Sprintf("%s: No skill in training", c.EveCharacter.Name)
		content := "There is currently no skill being trained for this character."
		notify(title, content)
		return nil, s.UpdateIsTrainingWatched(ctx, characterID, false)
	})
	if err != nil {
		return fmt.Errorf("NotifyExpiredTraining for character %d: %w", characterID, err)
	}
	return nil
}

// ListSkillqueueItems returns the list of skillqueue items.
func (s *CharacterService) ListSkillqueueItems(ctx context.Context, characterID int32) ([]*app.CharacterSkillqueueItem, error) {
	return s.st.ListCharacterSkillqueueItems(ctx, characterID)
}

// updateSkillqueueESI updates the skillqueue for a character from ESI
// and reports whether it has changed.
func (s *CharacterService) updateSkillqueueESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterSkillqueue {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
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
				_, err := s.eus.GetOrCreateTypeESI(ctx, o.SkillId)
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
