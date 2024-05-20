package model

type CharacterSkill struct {
	ActiveSkillLevel   int
	CharacterID        int32
	EveType            *EveType
	SkillPointsInSkill int
	TrainedSkillLevel  int
}

type ListCharacterSkillGroupProgress struct {
	ID      int32
	Name    string
	Total   float64
	Trained float64
}

type ListCharacterSkillProgress struct {
	ID                int32
	Description       string
	Name              string
	ActiveSkillLevel  int
	TrainedSkillLevel int
}
