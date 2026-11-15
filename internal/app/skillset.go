package app

// SkillSet is a named set of minimum active skill levels.
type SkillSet struct {
	ID           int64
	Name         string
	Requirements []SkillSetRequirement
}

// SkillSetRequirement is the minimum level for one skill in a skill set.
type SkillSetRequirement struct {
	TypeID int64
	Name   string
	Level  int
}
