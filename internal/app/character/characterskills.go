package character

import (
	"context"
	"errors"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) GetCharacterSkill(ctx context.Context, characterID, typeID int32) (*app.CharacterSkill, error) {
	o, err := s.st.GetCharacterSkill(ctx, characterID, typeID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	return o, err
}

func (s *CharacterService) updateCharacterSkillsESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionSkills {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			skills, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkills(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return skills, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			skills := data.(esi.GetCharactersCharacterIdSkillsOk)
			total := optional.New(int(skills.TotalSp))
			unallocated := optional.New(int(skills.UnallocatedSp))
			if err := s.st.UpdateCharacterSkillPoints(ctx, characterID, total, unallocated); err != nil {
				return err
			}
			x, err := s.st.ListCharacterSkillIDs(ctx, characterID)
			if err != nil {
				return err
			}
			currentSkillIDs := set.NewFromSlice(x)
			incomingSkillIDs := set.New[int32]()
			for _, o := range skills.Skills {
				incomingSkillIDs.Add(o.SkillId)
				_, err := s.EveUniverseService.GetOrCreateEveTypeESI(ctx, o.SkillId)
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
			obsoleteSkillIDs := currentSkillIDs.Difference(incomingSkillIDs)
			if obsoleteSkillIDs.Size() > 0 {
				if err := s.st.DeleteCharacterSkills(ctx, characterID, obsoleteSkillIDs.ToSlice()); err != nil {
					return err
				}
			}
			return nil
		})
}

func (s *CharacterService) ListCharacterSkillProgress(ctx context.Context, characterID, eveGroupID int32) ([]app.ListCharacterSkillProgress, error) {
	return s.st.ListCharacterSkillProgress(ctx, characterID, eveGroupID)
}

func (s *CharacterService) ListCharacterSkillGroupsProgress(ctx context.Context, characterID int32) ([]app.ListCharacterSkillGroupProgress, error) {
	return s.st.ListCharacterSkillGroupsProgress(ctx, characterID)
}
