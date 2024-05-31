package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

// CharacterSectionUpdatedAt returns when a section was last updated.
// It will return a zero time when no update has been completed yet.
func (s *Service) CharacterSectionUpdatedAt(characterID int32, section model.CharacterSection) (time.Time, error) {
	ctx := context.Background()
	u, err := s.r.GetCharacterUpdateStatus(ctx, characterID, section)
	if errors.Is(err, storage.ErrNotFound) {
		return time.Time{}, nil
	} else if err != nil {
		return time.Time{}, err
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
	case model.CharacterSectionAssets:
		f = s.updateCharacterAssetsESI
	case model.CharacterSectionAttributes:
		f = s.updateCharacterAttributesESI
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
		// TODO: Move this part into updateCharacterSectionIfChanged()
		errorMessage := humanize.Error(err)
		err2 := s.r.SetCharacterUpdateStatusError(ctx, characterID, section, errorMessage)
		if err2 != nil {
			slog.Error("failed to record error for failed section update: %s", err2)
		}
		s.characterStatus.SetError(characterID, section, errorMessage)
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
	if t.IsZero() {
		return true, nil
	}
	timeout := section.Timeout()
	deadline := t.Add(timeout)
	return time.Now().After(deadline), nil
}

// updateCharacterSectionIfChanged updates a character section if it has changed
// and reports wether it has changed
func (s *Service) updateCharacterSectionIfChanged(
	ctx context.Context,
	characterID int32,
	section model.CharacterSection,
	fetch func(ctx context.Context, characterID int32) (any, error),
	update func(ctx context.Context, characterID int32, data any) error,
) (bool, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	data, err := fetch(ctx, characterID)
	if err != nil {
		return false, err
	}
	// identify if changed
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

	// update if changed
	if hasChanged {
		if err := update(ctx, characterID, data); err != nil {
			return false, err
		}
	}

	// record update
	lastUpdatedAt := time.Now()
	arg := storage.CharacterUpdateStatusParams{
		CharacterID:   characterID,
		Section:       section,
		Error:         "",
		ContentHash:   hash,
		LastUpdatedAt: lastUpdatedAt,
	}
	if err := s.r.UpdateOrCreateCharacterUpdateStatus(ctx, arg); err != nil {
		return false, err
	}
	s.characterStatus.SetStatus(characterID, section, "", lastUpdatedAt)

	slog.Debug("Has section changed", "characterID", characterID, "section", section, "changed", hasChanged)
	return hasChanged, nil
}
