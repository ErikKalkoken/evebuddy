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
func timeoutForUpdateSection(section model.CharacterSection) time.Duration {
	m := map[model.CharacterSection]time.Duration{
		model.CharacterSectionHome:               120 * time.Second,
		model.CharacterSectionImplants:           120 * time.Second,
		model.CharacterSectionLocation:           30 * time.Second, // 5 seconds min
		model.CharacterSectionMailLabels:         30 * time.Second,
		model.CharacterSectionMailLists:          120 * time.Second,
		model.CharacterSectionMails:              30 * time.Second,
		model.CharacterSectionOnline:             60 * time.Second,
		model.CharacterSectionShip:               30 * time.Second, // 5 seconds min
		model.CharacterSectionSkillqueue:         120 * time.Second,
		model.CharacterSectionSkills:             120 * time.Second,
		model.CharacterSectionWalletBalance:      120 * time.Second,
		model.CharacterSectionWalletJournal:      3600 * time.Second,
		model.CharacterSectionWalletTransactions: 3600 * time.Second,
	}
	duration, ok := m[section]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", section)
		duration = defaultUpdateSectionTimeout
	}
	return duration
}

func (s *Service) CharacterSectionSetUpdated(characterID int32, section model.CharacterSection) error {
	ctx := context.Background()
	arg := storage.CharacterUpdateStatusParams{
		CharacterID: characterID,
		Section:     section,
		ContentHash: "",
		UpdatedAt:   time.Now(),
	}
	err := s.r.UpdateOrCreateCharacterUpdateStatus(ctx, arg)
	return err
}

// CharacterSectionUpdatedAt returns when a section was last updated.
// It will return a zero time when no update has been completed yet.
func (s *Service) CharacterSectionUpdatedAt(characterID int32, section model.CharacterSection) (time.Time, error) {
	ctx := context.Background()
	var zero time.Time
	u, err := s.r.GetCharacterUpdateStatus(ctx, characterID, section)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return zero, nil
		}
		return zero, err
	}
	return u.UpdatedAt, nil
}

// CharacterSectionWasUpdated reports wether the section has been updated at all.
func (s *Service) CharacterSectionWasUpdated(characterID int32, section model.CharacterSection) (bool, error) {
	t, err := s.CharacterSectionUpdatedAt(characterID, section)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return !t.IsZero(), nil
}

// UpdateCharacterSectionIfExpired updates a section from ESI if has expired and changed
// and reports back if it has changed
func (s *Service) UpdateCharacterSectionIfExpired(characterID int32, section model.CharacterSection) (bool, error) {
	isExpired, err := s.CharacterSectionIsUpdateExpired(characterID, section)
	if err != nil {
		return false, err
	}
	if !isExpired {
		return false, nil
	}
	ctx := context.Background()
	var f func(context.Context, int32) (bool, error)
	switch section {
	case model.CharacterSectionHome:
		f = s.updateCharacterHomeESI
	case model.CharacterSectionImplants:
		f = s.updateCharacterImplantsESI
	case model.CharacterSectionLocation:
		f = s.updateCharacterLocationESI
	case model.CharacterSectionMails:
		f = s.updateCharacterMailESI
	case model.CharacterSectionMailLabels:
		f = s.updateCharacterMailLabelsESI
	case model.CharacterSectionMailLists:
		f = s.updateCharacterMailListsESI
	case model.CharacterSectionOnline:
		f = s.updateCharacterOnlineESI
	case model.CharacterSectionShip:
		f = s.updateCharacterShipESI
	case model.CharacterSectionSkillqueue:
		f = s.updateCharacterSkillqueueESI
	case model.CharacterSectionSkills:
		f = s.updateCharacterSkillsESI
	case model.CharacterSectionWalletBalance:
		f = s.updateCharacterWalletBalanceESI
	case model.CharacterSectionWalletJournal:
		f = s.updateCharacterWalletJournalEntryESI
	case model.CharacterSectionWalletTransactions:
		f = s.updateCharacterWalletTransactionESI
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
	if err := s.CharacterSectionSetUpdated(characterID, section); err != nil {
		return false, err
	}
	changed := x.(bool)
	return changed, err
}

// SectionWasUpdated reports wether the data for a section has expired.
func (s *Service) CharacterSectionIsUpdateExpired(characterID int32, section model.CharacterSection) (bool, error) {
	t, err := s.CharacterSectionUpdatedAt(characterID, section)
	if err != nil {
		return false, err
	}
	timeout := timeoutForUpdateSection(section)
	deadline := t.Add(timeout)
	return time.Now().After(deadline), nil
}

// hasCharacterSectionChanged reports wether a section has changed based on the given data and updates it's content hash.
func (s *Service) hasCharacterSectionChanged(ctx context.Context, characterID int32, section model.CharacterSection, data any) (bool, error) {
	hash, err := calcHash(data)
	if err != nil {
		return false, err
	}
	u, err := s.r.GetCharacterUpdateStatus(ctx, characterID, section)
	if errors.Is(err, storage.ErrNotFound) {
		// section is new
	} else if err != nil {
		return false, err
	} else if u.ContentHash == hash {
		slog.Debug("Section has not changed", "characterID", characterID, "section", section)
		return false, nil
	}
	slog.Debug("Section has changed", "characterID", characterID, "section", section)
	arg := storage.CharacterUpdateStatusParams{
		CharacterID: characterID,
		Section:     section,
		ContentHash: hash,
		UpdatedAt:   time.Now(),
	}
	if err := s.r.UpdateOrCreateCharacterUpdateStatus(ctx, arg); err != nil {
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
