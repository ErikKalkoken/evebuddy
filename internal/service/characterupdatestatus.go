package service

import (
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

func (s *Service) CharacterListUpdateStatus(characterID int32) []model.CharacterStatus {
	list := make([]model.CharacterStatus, len(model.CharacterSections))
	for i, section := range model.CharacterSections {
		errorMessage, lastUpdatedAt := s.characterStatus.Get(characterID, section)
		list[i] = model.CharacterStatus{
			ErrorMessage:  errorMessage,
			LastUpdatedAt: lastUpdatedAt,
			Section:       section.Name(),
			Timeout:       section.Timeout(),
		}
	}
	return list
}

func (s *Service) CharacterGetUpdateStatusSummary() (float32, bool) {
	ids := s.characterStatus.GetCharacterIDs()
	total := len(model.CharacterSections) * len(ids)
	currentCount := 0
	for _, id := range ids {
		xx := s.CharacterListUpdateStatus(id)
		for _, x := range xx {
			if !x.IsOK() {
				return 0, false
			}
			if x.IsCurrent() {
				currentCount++
			}
		}
	}
	return float32(currentCount) / float32(total), true
}

func (s *Service) CharacterGetUpdateStatusCharacterSummary(characterID int32) (float32, bool) {
	total := len(model.CharacterSections)
	currentCount := 0
	xx := s.CharacterListUpdateStatus(characterID)
	for _, x := range xx {
		if !x.IsOK() {
			return 0, false
		}
		if x.IsCurrent() {
			currentCount++
		}
	}
	return float32(currentCount) / float32(total), true
}
