package service

import (
	"fmt"
	"log/slog"
	"time"
)

type UpdateSection string

const (
	UpdateSectionMail    = "mail"
	UpdateSectionDetails = "details"
)

func (s *Service) SectionSetNow(characterID int32, section UpdateSection) error {
	err := s.DictionarySetTime(makeUpdateAtDictKey(characterID, section), time.Now())
	return err
}

func (s *Service) SectionUpdatedAt(characterID int32, section UpdateSection) time.Time {
	t, err := s.DictionaryTime(makeUpdateAtDictKey(characterID, section))
	if err != nil {
		slog.Error(err.Error())
		return time.Time{}
	}
	return t
}

func makeUpdateAtDictKey(characterID int32, section UpdateSection) string {
	return fmt.Sprintf("%s-updated-at-%d", section, characterID)
}
