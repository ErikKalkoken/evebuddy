package app

import (
	"fmt"
	"strings"
)

// SkillPlan is a named set of minimum active skill levels.
type SkillPlan struct {
	ID           int64
	Name         string
	Requirements []SkillPlanRequirement
}

type SkillPlanRequirement struct {
	TypeID int64
	Name   string
	Level  int
}

func (p SkillPlan) Validate() error {
	if strings.TrimSpace(p.Name) == "" || len(p.Requirements) == 0 || p.ID < 0 {
		return fmt.Errorf("a plan needs a name and at least one skill: %w", ErrInvalid)
	}
	seen := make(map[int64]bool)
	for _, r := range p.Requirements {
		if r.TypeID <= 0 || r.Level < 1 || r.Level > 5 || seen[r.TypeID] {
			return fmt.Errorf("invalid or duplicate requirement for skill %d: %w", r.TypeID, ErrInvalid)
		}
		seen[r.TypeID] = true
	}
	return nil
}

type SkillPlanSkillStatus struct {
	Requirement SkillPlanRequirement
	ActiveLevel int64
}

func (s SkillPlanSkillStatus) Met() bool {
	return s.ActiveLevel >= int64(s.Requirement.Level)
}

// Evaluate compares active levels only; absent skills have level zero.
func (p SkillPlan) Evaluate(levels map[int64]int64) []SkillPlanSkillStatus {
	statuses := make([]SkillPlanSkillStatus, 0, len(p.Requirements))
	for _, r := range p.Requirements {
		statuses = append(statuses, SkillPlanSkillStatus{
			Requirement: r,
			ActiveLevel: levels[r.TypeID],
		})
	}
	return statuses
}
