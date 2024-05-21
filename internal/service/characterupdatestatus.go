package service

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

// CharacterSectionUpdatedAt returns when a section was last updated.
// It will return a zero time when no update has been completed yet.
func (s *Service) CharacterSectionUpdatedAt(characterID int32, section model.CharacterSection) (sql.NullTime, error) {
	ctx := context.Background()
	var zero sql.NullTime
	u, err := s.r.GetCharacterUpdateStatus(ctx, characterID, section)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return zero, nil
		}
		return zero, err
	}
	return u.LastUpdatedAt, nil
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
	return t.Valid, nil
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
	changed := x.(bool)
	return changed, err
}

// SectionWasUpdated reports wether the data for a section has expired.
func (s *Service) CharacterSectionIsUpdateExpired(characterID int32, section model.CharacterSection) (bool, error) {
	t, err := s.CharacterSectionUpdatedAt(characterID, section)
	if err != nil {
		return false, err
	}
	if !t.Valid {
		return true, nil
	}
	timeout := section.Timeout()
	deadline := t.Time.Add(timeout)
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
		CharacterID:   characterID,
		Section:       section,
		Error:         "",
		ContentHash:   hash,
		LastUpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
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

func (s *Service) ListCharacterUpdateStatus(characterID int32) ([]*model.CharacterUpdateStatus, error) {
	ctx := context.Background()
	return s.r.ListCharacterUpdateStatus(ctx, characterID)
}
