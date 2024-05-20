package model

type CharacterSkill struct {
	ActiveSkillLevel   int
	EveType            *EveType
	SkillPointsInSkill int
	MyCharacterID      int32
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
