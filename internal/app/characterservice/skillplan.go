package characterservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// ParseSkillPlan accepts one "Skill name level" per line, using I-V or 1-5.
// Repeated skills are merged at their highest requested level.
func ParseSkillPlan(name, text string, skills []*app.EveSkill) (app.SkillPlan, error) {
	p := app.SkillPlan{Name: strings.TrimSpace(name)}
	byName := make(map[string]*app.EveType)
	for _, s := range skills {
		byName[strings.ToLower(s.Type.Name)] = s.Type
	}
	byID := make(map[int64]app.SkillPlanRequirement)
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return p, fmt.Errorf("line %d: expected a skill name followed by a level: %w", i+1, app.ErrInvalid)
		}
		levelText := strings.ToUpper(fields[len(fields)-1])
		level := map[string]int{"I": 1, "II": 2, "III": 3, "IV": 4, "V": 5}[levelText]
		if level == 0 {
			level, _ = strconv.Atoi(levelText)
		}
		if level < 1 || level > 5 {
			return p, fmt.Errorf("line %d: level must be I-V or 1-5: %w", i+1, app.ErrInvalid)
		}
		skillName := strings.Join(fields[:len(fields)-1], " ")
		et, ok := byName[strings.ToLower(skillName)]
		if !ok {
			return p, fmt.Errorf("line %d: unknown skill %q; refresh the skill catalogue if needed: %w", i+1, skillName, app.ErrInvalid)
		}
		if old := byID[et.ID]; old.Level < level {
			byID[et.ID] = app.SkillPlanRequirement{TypeID: et.ID, Name: et.Name, Level: level}
		}
	}
	for _, r := range byID {
		p.Requirements = append(p.Requirements, r)
	}
	slices.SortFunc(p.Requirements, func(a, b app.SkillPlanRequirement) int {
		return strings.Compare(a.Name, b.Name)
	})
	return p, p.Validate()
}

func (s *CharacterService) SaveSkillPlan(ctx context.Context, id int64, name, text string) (int64, error) {
	skills, err := s.eus.ListSkills(ctx)
	if err != nil {
		return 0, err
	}
	p, err := ParseSkillPlan(name, text, skills)
	if err != nil {
		return 0, err
	}
	p.ID = id
	return s.st.SaveSkillPlan(ctx, p)
}

func (s *CharacterService) ListSkillPlans(ctx context.Context) ([]app.SkillPlan, error) {
	return s.st.ListSkillPlans(ctx)
}

func (s *CharacterService) DeleteSkillPlan(ctx context.Context, id int64) error {
	return s.st.DeleteSkillPlan(ctx, id)
}

type SkillPlanCharacter struct {
	Character   *app.EntityShort
	Tags        []string
	Skills      []app.SkillPlanSkillStatus
	SkillsKnown bool
	UpdatedAt   time.Time
	UpdateError string
}

func (c SkillPlanCharacter) MetCount() int {
	if !c.SkillsKnown {
		return 0
	}
	var n int
	for _, s := range c.Skills {
		if s.Met() {
			n++
		}
	}
	return n
}

func (c SkillPlanCharacter) Qualified() bool {
	return c.SkillsKnown && len(c.Skills) > 0 && c.MetCount() == len(c.Skills)
}

// CheckSkillPlan includes every character, even those with no skills or no tags.
// A character without a successful skills sync has unknown, not missing, skills.
func (s *CharacterService) CheckSkillPlan(ctx context.Context, p app.SkillPlan) ([]SkillPlanCharacter, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	characters, err := s.st.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	skills, err := s.st.ListAllCharacterSkills(ctx)
	if err != nil {
		return nil, err
	}
	levels := make(map[int64]map[int64]int64)
	for _, skill := range skills {
		if levels[skill.CharacterID] == nil {
			levels[skill.CharacterID] = make(map[int64]int64)
		}
		levels[skill.CharacterID][skill.Type.ID] = skill.ActiveSkillLevel
	}
	result := make([]SkillPlanCharacter, 0, len(characters))
	for _, character := range characters {
		tags, err := s.st.ListCharacterTagsForCharacter(ctx, character.ID)
		if err != nil {
			return nil, err
		}
		c := SkillPlanCharacter{Character: character, Skills: p.Evaluate(levels[character.ID])}
		for _, tag := range tags {
			c.Tags = append(c.Tags, tag.Name)
		}
		slices.Sort(c.Tags)
		status, err := s.st.GetCharacterSectionStatus(ctx, character.ID, app.SectionCharacterSkills)
		if err != nil && !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		if err == nil {
			c.SkillsKnown = !status.IsMissing()
			c.UpdatedAt = status.CompletedAt
			c.UpdateError = status.ErrorMessage
		}
		result = append(result, c)
	}
	slices.SortFunc(result, func(a, b SkillPlanCharacter) int {
		return cmp.Or(strings.Compare(a.Character.Name, b.Character.Name), cmp.Compare(a.Character.ID, b.Character.ID))
	})
	return result, nil
}
