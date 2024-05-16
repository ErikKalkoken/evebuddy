package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	ihttp "github.com/ErikKalkoken/evebuddy/internal/helper/http"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

// update section timeouts in seconds
const (
	updateSectionMailTimeout          = 30
	updateSectionDetailsTimeout       = 30
	updateSectionSkillqueueTimeout    = 120
	updateSectionWalletJournalTimeout = 3600
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
	case model.UpdateSectionMail:
		_, err := s.UpdateMailESI(characterID)
		if err != nil {
			return false, err
		}
		return true, nil

	case model.UpdateSectionSkillqueue:
		return s.UpdateSkillqueueESI(characterID)
	case model.UpdateSectionWalletJournal:
		return s.UpdateWalletJournalEntryESI(characterID)
	case model.UpdateSectionMyCharacter:
		err := s.UpdateMyCharacterESI(characterID)
		if err != nil {
			return false, err
		}
		return true, err
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

// hasSectionChanged reports wether a section has changed based on the given HTTP response.
func (s *Service) hasSectionChanged(ctx context.Context, characterID int32, section model.UpdateSection, r *http.Response) (bool, error) {
	hash := ihttp.CalcBodyHash(r)
	u, err := s.r.GetMyCharacterUpdateStatus(ctx, characterID, section)
	if errors.Is(err, storage.ErrNotFound) {
		// section is new
	} else if err != nil {
		return false, err
	} else if u.ContentHash == hash {
		slog.Debug("Section has not changed", "characterID", characterID, "section", section)
		return false, nil
	}
	slog.Debug("Section has changed", "characterID", characterID, "section", section)
	arg := storage.MyCharacterUpdateStatusParams{
		MyCharacterID: characterID,
		Section:       section,
		ContentHash:   hash,
		UpdatedAt:     time.Now(),
	}
	if err := s.r.UpdateOrCreateMyCharacterUpdateStatus(ctx, arg); err != nil {
		return false, err
	}
	return true, nil
}
