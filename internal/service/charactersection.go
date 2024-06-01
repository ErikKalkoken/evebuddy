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

// characterSectionUpdatedAt returns when a section was last updated.
// It will return a zero time when no update has been completed yet.
func (s *Service) characterSectionUpdatedAt(ctx context.Context, arg UpdateCharacterSectionParams) (time.Time, error) {
	u, err := s.r.GetCharacterUpdateStatus(ctx, arg.CharacterID, arg.Section)
	if errors.Is(err, storage.ErrNotFound) {
		return time.Time{}, nil
	} else if err != nil {
		return time.Time{}, err
	}
	return u.LastUpdatedAt, nil
}

// CharacterSectionWasUpdated reports wether the section has been updated at all.
func (s *Service) CharacterSectionWasUpdated(characterID int32, section model.CharacterSection) (bool, error) {
	ctx := context.Background()
	t, err := s.characterSectionUpdatedAt(ctx, UpdateCharacterSectionParams{CharacterID: characterID, Section: section})
	if errors.Is(err, storage.ErrNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return !t.IsZero(), nil
}

type UpdateCharacterSectionParams struct {
	CharacterID int32
	Section     model.CharacterSection
	ForceUpdate bool
}

// UpdateCharacterSection updates a section from ESI if has expired and changed
// and reports back if it has changed
func (s *Service) UpdateCharacterSection(arg UpdateCharacterSectionParams) (bool, error) {
	ctx := context.Background()
	if arg.CharacterID == 0 {
		panic("Invalid character ID")
	}
	if !arg.ForceUpdate {
		isExpired, err := s.characterSectionIsUpdateExpired(ctx, arg)
		if err != nil {
			return false, err
		}
		if !isExpired {
			return false, nil
		}
	}
	var f func(context.Context, UpdateCharacterSectionParams) (bool, error)
	switch arg.Section {
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
		f = s.updateCharacterMailsESI
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
		panic(fmt.Sprintf("Undefined section: %s", arg.Section))
	}
	key := fmt.Sprintf("UpdateESI-%s-%d", arg.Section, arg.CharacterID)
	x, err, _ := s.singleGroup.Do(key, func() (any, error) {
		return f(ctx, arg)
	})
	if err != nil {
		// TODO: Move this part into updateCharacterSectionIfChanged()
		errorMessage := humanize.Error(err)
		err2 := s.r.SetCharacterUpdateStatusError(ctx, arg.CharacterID, arg.Section, errorMessage)
		if err2 != nil {
			slog.Error("failed to record error for failed section update: %s", err2)
		}
		s.characterStatus.SetError(arg.CharacterID, arg.Section, errorMessage)
		return false, fmt.Errorf("failed to update character section from ESI for %v: %w", arg, err)
	}
	changed := x.(bool)
	return changed, err
}

// SectionWasUpdated reports wether the data for a section has expired.
func (s *Service) characterSectionIsUpdateExpired(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
	t, err := s.characterSectionUpdatedAt(ctx, arg)
	if err != nil {
		return false, err
	}
	if t.IsZero() {
		return true, nil
	}
	timeout := arg.Section.Timeout()
	deadline := t.Add(timeout)
	return time.Now().After(deadline), nil
}

// updateCharacterSectionIfChanged updates a character section if it has changed
// and reports wether it has changed
func (s *Service) updateCharacterSectionIfChanged(
	ctx context.Context,
	arg UpdateCharacterSectionParams,
	fetch func(ctx context.Context, characterID int32) (any, error),
	update func(ctx context.Context, characterID int32, data any) error,
) (bool, error) {
	token, err := s.getValidCharacterToken(ctx, arg.CharacterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	data, err := fetch(ctx, arg.CharacterID)
	if err != nil {
		return false, err
	}
	hash, err := arg.Section.CalcContentHash(data)
	if err != nil {
		return false, err
	}
	// identify if changed
	var hasChanged bool
	if !arg.ForceUpdate {
		u, err := s.r.GetCharacterUpdateStatus(ctx, arg.CharacterID, arg.Section)
		if errors.Is(err, storage.ErrNotFound) {
			hasChanged = true
		} else if err != nil {
			return false, err
		} else {
			hasChanged = u.ContentHash != hash
		}
	}
	// update if needed
	if arg.ForceUpdate || hasChanged {
		if err := update(ctx, arg.CharacterID, data); err != nil {
			return false, err
		}
	}

	// record update
	lastUpdatedAt := time.Now()
	arg2 := storage.CharacterUpdateStatusParams{
		CharacterID:   arg.CharacterID,
		Section:       arg.Section,
		Error:         "",
		ContentHash:   hash,
		LastUpdatedAt: lastUpdatedAt,
	}
	if err := s.r.UpdateOrCreateCharacterUpdateStatus(ctx, arg2); err != nil {
		return false, err
	}
	s.characterStatus.SetStatus(arg.CharacterID, arg.Section, "", lastUpdatedAt)

	slog.Debug("Has section changed", "characterID", arg.CharacterID, "section", arg.Section, "changed", hasChanged)
	return hasChanged, nil
}
