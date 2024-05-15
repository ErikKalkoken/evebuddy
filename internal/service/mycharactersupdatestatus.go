package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

// update section timeouts in seconds
const (
	updateSectionMailTimeout          = 600 // TODO: 60
	updateSectionDetailsTimeout       = 600 // TODO: 120
	updateSectionSkillqueueTimeout    = 600 // TODO: 120
	updateSectionWalletJournalTimeout = 600 // TODO: 120
)

func (s *Service) SectionSetUpdated(characterID int32, section model.UpdateSection) error {
	ctx := context.Background()
	arg := storage.MyCharacterUpdateStatusParams{
		MyCharacterID: characterID,
		Section:       section,
		ContentHash:   "",
		UpdatedAt:     time.Now(),
	}
	err := s.r.UpdateOrCreateMyCharacterUpdateStatus(ctx, arg)
	return err
}

// SectionUpdatedAt returns when a section was last updated.
// It will return a zero time when no update has been completed yet.
func (s *Service) SectionUpdatedAt(characterID int32, section model.UpdateSection) (time.Time, error) {
	ctx := context.Background()
	var zero time.Time
	u, err := s.r.GetMyCharacterUpdateStatus(ctx, characterID, section)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return zero, nil
		}
		return zero, err
	}
	return u.UpdatedAt, nil
}

// SectionWasUpdated reports wether the section has been updated at all.
func (s *Service) SectionWasUpdated(characterID int32, section model.UpdateSection) (bool, error) {
	t, err := s.SectionUpdatedAt(characterID, section)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return !t.IsZero(), nil
}

// SectionWasUpdated reports wether the data for a section has expired.
func (s *Service) SectionIsUpdateExpired(characterID int32, section model.UpdateSection) (bool, error) {
	t, err := s.SectionUpdatedAt(characterID, section)
	if err != nil {
		return false, err
	}
	deadline := t.Add(sectionUpdateTimeout(section))
	return time.Now().After(deadline), nil
}

func (s *Service) UpdateSectionIfExpired(characterID int32, section model.UpdateSection) (bool, error) {
	isExpired, err := s.SectionIsUpdateExpired(characterID, section)
	if err != nil {
		return false, err
	}
	if !isExpired {
		return false, nil
	}
	switch section {
	case model.UpdateSectionWalletJournal:
		return s.UpdateWalletJournalEntryESI(characterID)
	}
	panic(fmt.Sprintf("Undefined section: %s", section))
}

func sectionUpdateTimeout(section model.UpdateSection) time.Duration {
	m := map[model.UpdateSection]time.Duration{
		model.UpdateSectionMyCharacter:   updateSectionDetailsTimeout * time.Second,
		model.UpdateSectionMail:          updateSectionMailTimeout * time.Second,
		model.UpdateSectionSkillqueue:    updateSectionSkillqueueTimeout * time.Second,
		model.UpdateSectionWalletJournal: updateSectionWalletJournalTimeout * time.Second,
	}
	d, ok := m[section]
	if !ok {
		panic(fmt.Sprintf("Invalid section: %v", section))
	}
	return d
}
