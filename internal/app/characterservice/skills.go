package characterservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

const cacheKeyTrainingNotified = "expired-training-notified"

func (s *CharacterService) GetAttributes(ctx context.Context, characterID int64) (*app.CharacterAttributes, error) {
	return s.st.GetCharacterAttributes(ctx, characterID)
}

func (s *CharacterService) updateAttributesESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterAttributes {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdAttributes")
			attributes, _, err := s.esiClient.SkillsAPI.GetCharactersCharacterIdAttributes(ctx, characterID).Execute()
			if err != nil {
				return false, err
			}
			return attributes, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			attributes := data.(*esi.CharactersCharacterIdAttributesGet)
			err := s.st.UpdateOrCreateCharacterAttributes(ctx, storage.UpdateOrCreateCharacterAttributesParams{
				CharacterID:   characterID,
				BonusRemaps:   optional.FromPtr(attributes.BonusRemaps),
				Charisma:      attributes.Charisma,
				Intelligence:  attributes.Intelligence,
				LastRemapDate: optional.FromPtr(attributes.LastRemapDate),
				Memory:        attributes.Memory,
				Perception:    attributes.Perception,
				Willpower:     attributes.Willpower,
			})
			if err != nil {
				return false, err
			}
			return true, nil
		})
}

func (s *CharacterService) ListShipsAbilities(ctx context.Context, characterID int64) ([]*app.CharacterShipAbility, error) {
	return s.st.ListCharacterShipsAbilities(ctx, characterID)
}

func (s *CharacterService) GetSkill(ctx context.Context, characterID, typeID int64) (*app.CharacterSkill, error) {
	return s.st.GetCharacterSkill(ctx, characterID, typeID)
}

func (s *CharacterService) ListAllCharactersIndustrySlots(ctx context.Context, typ app.IndustryJobType) ([]app.CharacterIndustrySlots, error) {
	total := make(map[int64]int)
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
	results := make(map[int64]app.CharacterIndustrySlots)
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

func (s *CharacterService) ListSkills(ctx context.Context, characterID int64) ([]*app.CharacterSkill2, error) {
	oo, err := s.st.ListCharacterSkills(ctx, characterID)
	if err != nil {
		return nil, err
	}
	skills := maps.Collect(xiter.MapSlice2(oo, func(x *app.CharacterSkill) (int64, *app.CharacterSkill) {
		return x.Type.ID, x
	}))
	eveSkills, err := s.eus.ListSkills(ctx)
	if err != nil {
		return nil, err
	}
	var skills2 []*app.CharacterSkill2
	for _, es := range eveSkills {
		o := &app.CharacterSkill2{
			CharacterID: characterID,
			Skill:       es,
		}
		if s, ok := skills[es.Type.ID]; ok {
			o.ActiveSkillLevel = s.ActiveSkillLevel
			o.TrainedSkillLevel = s.TrainedSkillLevel
			o.SkillPointsInSkill = s.SkillPointsInSkill
		}
		hasPrerequisites := true
		for _, r := range es.Requirements {
			s2, ok := skills[r.Type.ID]
			if !ok {
				hasPrerequisites = false
				break
			}
			if r.Level > int(s2.ActiveSkillLevel) {
				hasPrerequisites = false
				break
			}
		}
		o.HasPrerequisites = hasPrerequisites
		skills2 = append(skills2, o)
	}
	return skills2, nil
}

func (s *CharacterService) updateSkillsESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterSkills {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdSkills")
			skills, _, err := s.esiClient.SkillsAPI.GetCharactersCharacterIdSkills(ctx, characterID).Execute()
			if err != nil {
				return false, err
			}
			slog.Debug("Received character skills from ESI", "characterID", characterID, "items", len(skills.Skills))
			return skills, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			skills := data.(*esi.CharactersSkills)
			total := optional.New(skills.TotalSp)
			unallocated := optional.FromPtr(skills.UnallocatedSp)
			if err := s.st.UpdateCharacterSkillPoints(ctx, characterID, total, unallocated); err != nil {
				return false, err
			}
			currentSkillIDs, err := s.st.ListCharacterSkillIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			incomingSkillIDs := set.Of[int64]()
			for _, o := range skills.Skills {
				incomingSkillIDs.Add(o.SkillId)
				_, err := s.eus.GetOrCreateTypeESI(ctx, o.SkillId)
				if err != nil {
					return false, err
				}
				arg := storage.UpdateOrCreateCharacterSkillParams{
					CharacterID:        characterID,
					TypeID:             o.SkillId,
					ActiveSkillLevel:   o.ActiveSkillLevel,
					TrainedSkillLevel:  o.TrainedSkillLevel,
					SkillPointsInSkill: o.SkillpointsInSkill,
				}
				err = s.st.UpdateOrCreateCharacterSkill(ctx, arg)
				if err != nil {
					return false, err
				}
			}
			slog.Info("Stored updated character skills", "characterID", characterID, "count", len(skills.Skills))
			if ids := set.Difference(currentSkillIDs, incomingSkillIDs); ids.Size() > 0 {
				if err := s.st.DeleteCharacterSkills(ctx, characterID, ids); err != nil {
					return false, err
				}
				slog.Info("Deleted obsolete character skills", "characterID", characterID, "count", ids.Size())
			}
			return true, nil
		})
}

// TotalTrainingTime returns the total training time for a character when available.
// A training time of 0 means that training is not active.
// An empty training time means that the training status could not be determined.
func (s *CharacterService) TotalTrainingTime(ctx context.Context, characterID int64) (optional.Optional[time.Duration], error) {
	var z optional.Optional[time.Duration]
	v, err := s.st.GetCharacterSectionStatus(ctx, characterID, app.SectionCharacterSkillqueue)
	if errors.Is(err, app.ErrNotFound) {
		return z, nil
	}
	if err != nil {
		return z, err
	}
	if !isValidSkillQueueStatus(v) {
		return z, nil
	}
	d, err := s.st.GetCharacterTotalTrainingTime(ctx, characterID)
	if err != nil {
		return z, err
	}
	return optional.New(d), nil
}

func isValidSkillQueueStatus(v *app.CharacterSectionStatus) bool {
	if v.CompletedAt.IsZero() || v.HasError() {
		return false
	}
	stale := v.CompletedAt.Add(12 * time.Hour)
	return time.Now().Before(stale)
}

// IsTrainingActive reports whether training is active.
func (s *CharacterService) IsTrainingActive(ctx context.Context, characterID int64) (bool, error) {
	queue := app.NewCharacterSkillqueue()
	if err := queue.Update(ctx, s, characterID); err != nil {
		return false, err
	}
	return queue.IsActive(), nil
}

func (s *CharacterService) UpdateIsTrainingWatched(ctx context.Context, characterID int64, v bool) error {
	s.cache.Delete(makeKeyTrainingNotified(characterID))
	return s.st.UpdateCharacterIsTrainingWatched(ctx, characterID, v)
}

func (s *CharacterService) NotifyExpiredTrainingForWatched(ctx context.Context, characterID int64, notify func(title, content string)) error {
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
		key := makeKeyTrainingNotified(characterID)
		_, found := s.cache.GetString(key)
		if found {
			return nil, nil
		}
		title := fmt.Sprintf("%s: No skill in training", c.EveCharacter.Name)
		content := fmt.Sprintf(
			"There is currently no skill in trained for the watched character %s.",
			c.EveCharacter.Name,
		)
		notify(title, content)
		s.cache.SetString(key, title, 24*time.Hour)
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("NotifyExpiredTraining for character %d: %w", characterID, err)
	}
	return nil
}

func makeKeyTrainingNotified(characterID int64) string {
	return fmt.Sprintf("%s-%d", cacheKeyTrainingNotified, characterID)
}

// ListSkillqueueItems returns the list of skillqueue items.
func (s *CharacterService) ListSkillqueueItems(ctx context.Context, characterID int64) ([]*app.CharacterSkillqueueItem, error) {
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
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdSkillqueue")
			items, _, err := s.esiClient.SkillsAPI.GetCharactersCharacterIdSkillqueue(ctx, characterID).Execute()
			if err != nil {
				return false, err
			}
			slog.Debug("Received skillqueue from ESI", "characterID", characterID, "items", len(items))
			return items, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			items := make([]storage.SkillqueueItemParams, 0)
			for _, o := range data.([]esi.CharactersSkillqueueSkill) {
				if o.SkillId == 0 || o.FinishedLevel == 0 {
					continue
				}
				_, err := s.eus.GetOrCreateTypeESI(ctx, o.SkillId)
				if err != nil {
					return false, err
				}
				items = append(items, storage.SkillqueueItemParams{
					EveTypeID:       o.SkillId,
					FinishDate:      optional.FromPtr(o.FinishDate),
					FinishedLevel:   o.FinishedLevel,
					LevelEndSP:      optional.FromPtr(o.LevelEndSp),
					LevelStartSP:    optional.FromPtr(o.LevelStartSp),
					CharacterID:     characterID,
					QueuePosition:   o.QueuePosition,
					StartDate:       optional.FromPtr(o.StartDate),
					TrainingStartSP: optional.FromPtr(o.TrainingStartSp),
				})
			}
			if err := s.st.ReplaceCharacterSkillqueueItems(ctx, characterID, items); err != nil {
				return false, err
			}
			slog.Info("Stored updated skillqueue items", "characterID", characterID, "count", len(items))
			return true, nil
		})

}
