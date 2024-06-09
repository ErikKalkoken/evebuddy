package character

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	igoesi "github.com/ErikKalkoken/evebuddy/internal/helper/goesi"
	"github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

// sectionUpdatedAt returns when a section was last updated.
// It will return a zero time when no update has been completed yet.
func (s *CharacterService) sectionUpdatedAt(ctx context.Context, arg UpdateSectionParams) (time.Time, error) {
	u, err := s.st.GetCharacterUpdateStatus(ctx, arg.CharacterID, arg.Section)
	if errors.Is(err, storage.ErrNotFound) {
		return time.Time{}, nil
	} else if err != nil {
		return time.Time{}, err
	}
	return u.CompletedAt, nil
}

// SectionWasUpdated reports wether the section has been updated at all.
func (s *CharacterService) SectionWasUpdated(ctx context.Context, characterID int32, section model.CharacterSection) (bool, error) {
	t, err := s.sectionUpdatedAt(ctx, UpdateSectionParams{CharacterID: characterID, Section: section})
	if errors.Is(err, storage.ErrNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return !t.IsZero(), nil
}

type UpdateSectionParams struct {
	CharacterID int32
	Section     model.CharacterSection
	ForceUpdate bool
}

// UpdateSection updates a section from ESI if has expired and changed
// and reports back if it has changed
func (s *CharacterService) UpdateSection(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.CharacterID == 0 {
		panic("Invalid character ID")
	}
	if !arg.ForceUpdate {
		isExpired, err := s.sectionIsUpdateExpired(ctx, arg)
		if err != nil {
			return false, err
		}
		if !isExpired {
			return false, nil
		}
	}
	var f func(context.Context, UpdateSectionParams) (bool, error)
	switch arg.Section {
	case model.SectionAssets:
		f = s.updateCharacterAssetsESI
	case model.SectionAttributes:
		f = s.updateCharacterAttributesESI
	case model.SectionImplants:
		f = s.updateCharacterImplantsESI
	case model.SectionJumpClones:
		f = s.updateCharacterJumpClonesESI
	case model.SectionLocation:
		f = s.updateCharacterLocationESI
	case model.SectionMails:
		f = s.updateCharacterMailsESI
	case model.SectionMailLabels:
		f = s.updateCharacterMailLabelsESI
	case model.SectionMailLists:
		f = s.updateCharacterMailListsESI
	case model.SectionOnline:
		f = s.updateCharacterOnlineESI
	case model.SectionShip:
		f = s.updateCharacterShipESI
	case model.SectionSkillqueue:
		f = s.UpdateCharacterSkillqueueESI
	case model.SectionSkills:
		f = s.updateCharacterSkillsESI
	case model.SectionWalletBalance:
		f = s.updateCharacterWalletBalanceESI
	case model.SectionWalletJournal:
		f = s.updateCharacterWalletJournalEntryESI
	case model.SectionWalletTransactions:
		f = s.updateCharacterWalletTransactionESI
	default:
		panic(fmt.Sprintf("Undefined section: %s", arg.Section))
	}
	key := fmt.Sprintf("UpdateESI-%s-%d", arg.Section, arg.CharacterID)
	x, err, _ := s.sfg.Do(key, func() (any, error) {
		return f(ctx, arg)
	})
	if err != nil {
		// TODO: Move this part into updateCharacterSectionIfChanged()
		errorMessage := humanize.Error(err)
		opt := storage.CharacterUpdateStatusOptionals{
			Error: storage.NewNullString(errorMessage),
		}
		err2 := s.st.UpdateOrCreateCharacterUpdateStatus2(ctx, arg.CharacterID, arg.Section, opt)
		if err2 != nil {
			slog.Error("failed to record error for failed section update: %s", err2)
		}
		s.cs.SetError(arg.CharacterID, arg.Section, errorMessage)
		return false, fmt.Errorf("failed to update character section from ESI for %v: %w", arg, err)
	}
	changed := x.(bool)
	return changed, err
}

// SectionWasUpdated reports wether the data for a section has expired.
func (s *CharacterService) sectionIsUpdateExpired(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	t, err := s.sectionUpdatedAt(ctx, arg)
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

// updateSectionIfChanged updates a character section if it has changed
// and reports wether it has changed
func (s *CharacterService) updateSectionIfChanged(
	ctx context.Context,
	arg UpdateSectionParams,
	fetch func(ctx context.Context, characterID int32) (any, error),
	update func(ctx context.Context, characterID int32, data any) error,
) (bool, error) {
	token, err := s.getValidCharacterToken(ctx, arg.CharacterID)
	if err != nil {
		return false, err
	}
	ctx = igoesi.ContextWithESIToken(ctx, token.AccessToken)
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
		u, err := s.st.GetCharacterUpdateStatus(ctx, arg.CharacterID, arg.Section)
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
	completedAt := time.Now()
	arg2 := storage.CharacterUpdateStatusParams{
		CharacterID: arg.CharacterID,
		Section:     arg.Section,
		Error:       "",
		ContentHash: hash,
		CompletedAt: completedAt,
	}
	if err := s.st.UpdateOrCreateCharacterUpdateStatus(ctx, arg2); err != nil {
		return false, err
	}
	s.cs.SetStatus(arg.CharacterID, arg.Section, "", completedAt)

	slog.Debug("Has section changed", "characterID", arg.CharacterID, "section", arg.Section, "changed", hasChanged)
	return hasChanged, nil
}
