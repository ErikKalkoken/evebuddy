package service

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

type CharacterUpdateStatus2 struct {
	ErrorMessage  string
	LastUpdatedAt time.Time
	Section       string
	Timeout       time.Duration
}

func (s *CharacterUpdateStatus2) IsOK() bool {
	return s.ErrorMessage == ""
}

func (s *CharacterUpdateStatus2) IsCurrent() bool {
	if s.LastUpdatedAt.IsZero() {
		return false
	}
	return time.Now().Before(s.LastUpdatedAt.Add(s.Timeout * 2))
}

func (s *Service) CharacterListUpdateStatus(characterID int32) []CharacterUpdateStatus2 {
	list := make([]CharacterUpdateStatus2, len(model.CharacterSections))
	for i, section := range model.CharacterSections {
		errorMessage, lastUpdatedAt := s.statusCache.getStatus(characterID, section)
		list[i] = CharacterUpdateStatus2{
			ErrorMessage:  errorMessage,
			LastUpdatedAt: lastUpdatedAt,
			Section:       section.Name(),
			Timeout:       section.Timeout(),
		}
	}
	return list
}

func (s *Service) CharacterGetUpdateStatusSummary() (float32, bool) {
	ids := s.statusCache.getCharacterIDs()
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
