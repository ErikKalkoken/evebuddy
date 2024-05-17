package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

const defaultUpdateSectionTimeout = 3600 * time.Second

// timeoutForUpdateSection returns the time until the data of an update section becomes stale.
func timeoutForUpdateSection(section model.UpdateSection) time.Duration {
	m := map[model.UpdateSection]time.Duration{
		model.UpdateSectionHome:          120 * time.Second,
		model.UpdateSectionLocation:      30 * time.Second, // 5 seconds min
		model.UpdateSectionMailLabels:    30 * time.Second,
		model.UpdateSectionMailLists:     120 * time.Second,
		model.UpdateSectionMails:         30 * time.Second,
		model.UpdateSectionOnline:        60 * time.Second,
		model.UpdateSectionShip:          30 * time.Second, // 5 seconds min
		model.UpdateSectionSkillqueue:    120 * time.Second,
		model.UpdateSectionSkills:        120 * time.Second,
		model.UpdateSectionWalletBalance: 120 * time.Second,
		model.UpdateSectionWalletJournal: 3600 * time.Second,
	}
	duration, ok := m[section]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", section)
		duration = defaultUpdateSectionTimeout
	}
	return duration
}

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

// UpdateSectionIfExpired updates a section from ESI if has expired and changed
// and reports back if it has changed
func (s *Service) UpdateSectionIfExpired(characterID int32, section model.UpdateSection) (bool, error) {
	isExpired, err := s.SectionIsUpdateExpired(characterID, section)
	if err != nil {
		return false, err
	}
	if !isExpired {
		return false, nil
	}
	ctx := context.Background()
	var f func(context.Context, int32) (bool, error)
	switch section {
	case model.UpdateSectionHome:
		f = s.updateHomeESI
	case model.UpdateSectionLocation:
		f = s.updateLocationESI
	case model.UpdateSectionMails:
		f = s.updateMailESI
	case model.UpdateSectionMailLabels:
		f = s.updateMailLabelsESI
	case model.UpdateSectionMailLists:
		f = s.updateMailListsESI
	case model.UpdateSectionOnline:
		f = s.updateOnlineESI
	case model.UpdateSectionShip:
		f = s.updateShipESI
	case model.UpdateSectionSkillqueue:
		f = s.updateSkillqueueESI
	case model.UpdateSectionSkills:
		f = s.updateSkillsESI
	case model.UpdateSectionWalletBalance:
		f = s.updateWalletBalanceESI
	case model.UpdateSectionWalletJournal:
		f = s.updateWalletJournalEntryESI
	default:
		panic(fmt.Sprintf("Undefined section: %s", section))
	}
	key := fmt.Sprintf("UpdateESI-%s-%d", section, characterID)
	x, err, _ := s.singleGroup.Do(key, func() (any, error) {
		return f(ctx, characterID)
	})
	if err != nil {
		return false, fmt.Errorf("failed to update section %s from ESI for character %d: %w", section, characterID, err)
	}
	if err := s.SectionSetUpdated(characterID, section); err != nil {
		return false, err
	}
	changed := x.(bool)
	return changed, err
}

// SectionWasUpdated reports wether the data for a section has expired.
func (s *Service) SectionIsUpdateExpired(characterID int32, section model.UpdateSection) (bool, error) {
	t, err := s.SectionUpdatedAt(characterID, section)
	if err != nil {
		return false, err
	}
	duration := timeoutForUpdateSection(section)
	deadline := t.Add(duration)
	return time.Now().After(deadline), nil
}

// TODO: Need to change the API to accept any data

// hasSectionChanged reports wether a section has changed based on the given data and updates it's content hash.
func (s *Service) hasSectionChanged(ctx context.Context, characterID int32, section model.UpdateSection, data any) (bool, error) {
	hash, err := calcHash(data)
	if err != nil {
		return false, err
	}
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

func calcHash(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	b2 := md5.Sum(b)
	hash := hex.EncodeToString(b2[:])
	return hash, nil
}
