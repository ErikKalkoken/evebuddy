package service

import (
	"fmt"
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

// SectionUpdatedAt returns when a section was last updated.
// It will return a zero time when no update has been completed yet.
func (s *Service) SectionUpdatedAt(characterID int32, section UpdateSection) (time.Time, error) {
	var zero time.Time
	t, ok, err := s.DictionaryTime(makeUpdateAtDictKey(characterID, section))
	if err != nil {
		return zero, err
	}
	if !ok {
		return zero, nil
	}
	return t, nil
}

// SectionWasUpdated reports wether the section has been updated at all.
func (s *Service) SectionWasUpdated(characterID int32, section UpdateSection) (bool, error) {
	t, err := s.SectionUpdatedAt(characterID, section)
	if err != nil {
		return false, err
	}
	return !t.IsZero(), nil
}

// SectionWasUpdated reports wether the data for a section has expired.
func (s *Service) SectionIsUpdateExpired(characterID int32, section UpdateSection) (bool, error) {
	t, err := s.SectionUpdatedAt(characterID, section)
	if err != nil {
		return false, err
	}
	deadline := t.Add(sectionUpdateTimeout(section))
	return time.Now().After(deadline), nil
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
