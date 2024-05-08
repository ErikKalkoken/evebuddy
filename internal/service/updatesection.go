package service

import (
	"fmt"
	"log/slog"
	"time"
)

type UpdateSection string

// Updated character sections
const (
	UpdateSectionMail          = "mail"
	UpdateSectionMyCharacter   = "my_character"
	UpdateSectionSkillqueue    = "skillqueue"
	UpdateSectionWalletJournal = "wallet_journal"
)

// update section timeouts in seconds
const (
	updateSectionMailTimeout          = 600 // TODO: 60
	updateSectionDetailsTimeout       = 600 // TODO: 120
	updateSectionSkillqueueTimeout    = 600 // TODO: 120
	updateSectionWalletJournalTimeout = 600 // TODO: 120
)

func (s *Service) SectionSetUpdated(characterID int32, section UpdateSection) error {
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

func (s *Service) SectionUpdatedExpired(characterID int32, section UpdateSection) bool {
	deadline := s.SectionUpdatedAt(characterID, section).Add(sectionUpdateTimeout(section))
	return time.Now().After(deadline)
}

func makeUpdateAtDictKey(characterID int32, section UpdateSection) string {
	return fmt.Sprintf("%s-updated-at-%d", section, characterID)
}

func sectionUpdateTimeout(section UpdateSection) time.Duration {
	m := map[UpdateSection]time.Duration{
		UpdateSectionMyCharacter:   updateSectionDetailsTimeout * time.Second,
		UpdateSectionMail:          updateSectionMailTimeout * time.Second,
		UpdateSectionSkillqueue:    updateSectionSkillqueueTimeout * time.Second,
		UpdateSectionWalletJournal: updateSectionWalletJournalTimeout * time.Second,
	}
	d, ok := m[section]
	if !ok {
		panic(fmt.Sprintf("Invalid section: %v", section))
	}
	return d
}
