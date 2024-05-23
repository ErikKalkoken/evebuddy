package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi/esi"
)

// CharacterSectionUpdatedAt returns when a section was last updated.
// It will return a zero time when no update has been completed yet.
func (s *Service) CharacterSectionUpdatedAt(characterID int32, section model.CharacterSection) (sql.NullTime, error) {
	ctx := context.Background()
	var zero sql.NullTime
	u, err := s.r.GetCharacterUpdateStatus(ctx, characterID, section)
	if errors.Is(err, storage.ErrNotFound) {
		return zero, nil
	} else if err != nil {
		return zero, err
	}
	return u.LastUpdatedAt, nil
}

// CharacterSectionWasUpdated reports wether the section has been updated at all.
func (s *Service) CharacterSectionWasUpdated(characterID int32, section model.CharacterSection) (bool, error) {
	t, err := s.CharacterSectionUpdatedAt(characterID, section)
	if errors.Is(err, storage.ErrNotFound) {
		return false, nil
	} else if err != nil {
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
	case model.CharacterSectionAttributes:
		f = s.updateCharacterAttributesESI
	case model.CharacterSectionHome:
		f = s.updateCharacterHomeESI
	case model.CharacterSectionImplants:
		f = s.updateCharacterImplantsESI
	case model.CharacterSectionJumpClones:
		f = s.updateCharacterJumpClonesESI
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
		t := err.Error()
		e1, ok := err.(esi.GenericSwaggerError)
		if ok {
			e2, ok := e1.Model().(esi.InternalServerError)
			if ok {
				t += ": " + e2.Error_
			}
		}
		err2 := s.r.SetCharacterUpdateStatusError(ctx, characterID, section, t)
		if err2 != nil {
			slog.Error("failed to record error for failed section update: %s", err2)
		}
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

// recordCharacterSectionUpdate records an update to a character section
// and reports wether the section has changed
func (s *Service) recordCharacterSectionUpdate(ctx context.Context, characterID int32, section model.CharacterSection, data any) (bool, error) {
	hash, err := section.CalcContentHash(data)
	if err != nil {
		return false, err
	}
	var hasChanged bool
	u, err := s.r.GetCharacterUpdateStatus(ctx, characterID, section)
	if errors.Is(err, storage.ErrNotFound) {
		hasChanged = true
	} else if err != nil {
		return false, err
	} else {
		hasChanged = u.ContentHash != hash
	}
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
	slog.Debug("Has section changed", "characterID", characterID, "section", section, "changed", hasChanged)
	return hasChanged, nil
}

func (s *Service) ListCharacterUpdateStatus(characterID int32) ([]*model.CharacterUpdateStatus, error) {
	ctx := context.Background()
	return s.r.ListCharacterUpdateStatus(ctx, characterID)
}
