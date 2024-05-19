package model

type CharacterSkill struct {
	ActiveSkillLevel   int
	EveType            *EveType
	SkillPointsInSkill int
	MyCharacterID      int32
	TrainedSkillLevel  int
}
