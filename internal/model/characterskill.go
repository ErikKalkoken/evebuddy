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
	Total   int
	Trained int
}
