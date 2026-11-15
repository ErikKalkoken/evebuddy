package characterservice

import (
	"context"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// CreateSkillSet creates a skill set and returns its ID.
// Returns [app.ErrAlreadyExists] when the name is taken.
func (s *CharacterService) CreateSkillSet(ctx context.Context, skillSet app.SkillSet) (int64, error) {
	if err := s.eus.AddMissingTypes(ctx, skillSetTypeIDs(skillSet)); err != nil {
		return 0, err
	}
	return s.st.CreateSkillSet(ctx, skillSet)
}

// UpdateSkillSet updates the name and replaces all requirements of a skill set.
// Returns [app.ErrNotFound] when the skill set does not exist.
func (s *CharacterService) UpdateSkillSet(ctx context.Context, skillSet app.SkillSet) error {
	if err := s.eus.AddMissingTypes(ctx, skillSetTypeIDs(skillSet)); err != nil {
		return err
	}
	return s.st.UpdateSkillSet(ctx, skillSet)
}

func skillSetTypeIDs(skillSet app.SkillSet) set.Set[int64] {
	var ids set.Set[int64]
	for _, r := range skillSet.Requirements {
		ids.Add(r.TypeID)
	}
	return ids
}

// ListSkillSets returns all skill sets ordered by name, with requirements ordered by skill name.
func (s *CharacterService) ListSkillSets(ctx context.Context) ([]app.SkillSet, error) {
	return s.st.ListSkillSets(ctx)
}

// DeleteSkillSet deletes a skill set.
// Returns [app.ErrNotFound] when the skill set does not exist.
func (s *CharacterService) DeleteSkillSet(ctx context.Context, id int64) error {
	return s.st.DeleteSkillSet(ctx, id)
}
