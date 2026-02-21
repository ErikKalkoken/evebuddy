package eveuniverseservice

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func (s *EveUniverseService) ListSkillGroups(ctx context.Context) ([]*app.EveSkillGroup, error) {
	return s.st.ListEveSkillGroups(ctx)
}

var skillDogmaAttributes = []struct {
	typeID int64
	level  int64
}{
	{app.EveDogmaAttributePrimarySkillID, app.EveDogmaAttributePrimarySkillLevel},
	{app.EveDogmaAttributeSecondarySkillID, app.EveDogmaAttributeSecondarySkillLevel},
	{app.EveDogmaAttributeTertiarySkillID, app.EveDogmaAttributeTertiarySkillLevel},
	{app.EveDogmaAttributeQuaternarySkillID, app.EveDogmaAttributeQuaternarySkillLevel},
	{app.EveDogmaAttributeQuinarySkillID, app.EveDogmaAttributeQuinarySkillLevel},
	{app.EveDogmaAttributeSenarySkillID, app.EveDogmaAttributeSenarySkillLevel},
}

func (s *EveUniverseService) ListSkills(ctx context.Context) ([]*app.EveSkill, error) {
	es, err := s.st.ListEveSkills(ctx)
	if err != nil {
		return nil, err
	}
	types := make(map[int64]*app.EveType)
	for _, et := range es {
		types[et.ID] = et
	}
	da, err := s.st.ListEveTypeDogmaAttributesForSkills(ctx)
	if err != nil {
		return nil, err
	}
	attributes := make(map[int64]map[int64]float64)
	for _, o := range da {
		if attributes[o.Type.ID] == nil {
			attributes[o.Type.ID] = make(map[int64]float64)
		}
		attributes[o.Type.ID][o.DogmaAttribute.ID] = o.Value
	}

	var skills []*app.EveSkill
	for _, et := range types {
		var sp, rank optional.Optional[int]
		if x, ok := attributes[et.ID][app.EveDogmaAttributeTrainingTimeMultiplier]; ok {
			rank.Set(int(x))
			sp.Set(256_000 * int(x))
		}
		skill := &app.EveSkill{
			Rank:         rank,
			Requirements: make(map[int]*app.EveRequiredSkill),
			Skillpoints:  sp,
			Type:         et,
		}
		for rank, x := range skillDogmaAttributes {
			typeID, ok := attributes[et.ID][x.typeID]
			if !ok {
				continue
			}
			level, ok := attributes[et.ID][x.level]
			if !ok {
				continue
			}
			skill.Requirements[rank] = &app.EveRequiredSkill{
				Type:  types[int64(typeID)],
				Level: int(level),
			}
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

func (s *EveUniverseService) UpdateShipSkills(ctx context.Context) error {
	return s.st.UpdateEveShipSkills(ctx)
}
