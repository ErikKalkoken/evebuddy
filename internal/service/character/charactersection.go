package character

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

	igoesi "github.com/ErikKalkoken/evebuddy/internal/helper/goesi"
	"github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

// SectionWasUpdated reports wether the section has been updated at all.
func (s *CharacterService) SectionWasUpdated(ctx context.Context, characterID int32, section model.CharacterSection) (bool, error) {
	status, err := s.getCharacterSectionStatus(ctx, characterID, section)
	if err != nil {
		return false, err
	}
	if status == nil {
		return false, nil
	}
	return !status.CompletedAt.IsZero(), nil
}

type UpdateSectionParams struct {
	CharacterID int32
	Section     model.CharacterSection
	ForceUpdate bool
}

// UpdateSectionIfNeeded updates a section from ESI if has expired and changed
// and reports back if it has changed
func (s *CharacterService) UpdateSectionIfNeeded(ctx context.Context, arg UpdateSectionParams) (bool, error) {
	if arg.CharacterID == 0 {
		panic("Invalid character ID")
	}
	if !arg.ForceUpdate {
		status, err := s.getCharacterSectionStatus(ctx, arg.CharacterID, arg.Section)
		if err != nil {
			return false, err
		}
		if status != nil {
			if status.IsOK() && !status.IsExpired() {
				return false, nil
			}
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
		startedAt := sql.NullTime{}
		opt := storage.CharacterSectionStatusOptionals{Error: &errorMessage, StartedAt: &startedAt}
		o, err2 := s.st.UpdateOrCreateCharacterSectionStatus2(ctx, arg.CharacterID, arg.Section, opt)
		if err2 != nil {
			slog.Error("failed to record error for failed section update: %s", err2)
		}
		s.cs.CharacterSet(o)
		return false, fmt.Errorf("failed to update character section from ESI for %v: %w", arg, err)
	}
	changed := x.(bool)
	return changed, err
}

// updateSectionIfChanged updates a character section if it has changed
// and reports wether it has changed
func (s *CharacterService) updateSectionIfChanged(
	ctx context.Context,
	arg UpdateSectionParams,
	fetch func(ctx context.Context, characterID int32) (any, error),
	update func(ctx context.Context, characterID int32, data any) error,
) (bool, error) {
	startedAt := storage.NewNullTime(time.Now())
	opt := storage.CharacterSectionStatusOptionals{
		StartedAt: &startedAt,
	}
	o, err := s.st.UpdateOrCreateCharacterSectionStatus2(ctx, arg.CharacterID, arg.Section, opt)
	if err != nil {
		return false, err
	}
	s.cs.CharacterSet(o)
	token, err := s.getValidCharacterToken(ctx, arg.CharacterID)
	if err != nil {
		return false, err
	}
	ctx = igoesi.ContextWithESIToken(ctx, token.AccessToken)
	data, err := fetch(ctx, arg.CharacterID)
	if err != nil {
		return false, err
	}
	hash, err := calcContentHash(data)
	if err != nil {
		return false, err
	}
	// identify if changed
	var hasChanged bool
	if !arg.ForceUpdate {
		u, err := s.getCharacterSectionStatus(ctx, arg.CharacterID, arg.Section)
		if err != nil {
			return false, err
		}
		if u == nil {
			hasChanged = true
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

	// record successful completion
	completedAt := storage.NewNullTime(time.Now())
	errorMessage := ""
	startedAt2 := sql.NullTime{}
	opt = storage.CharacterSectionStatusOptionals{
		Error:       &errorMessage,
		ContentHash: &hash,
		CompletedAt: &completedAt,
		StartedAt:   &startedAt2,
	}
	o, err = s.st.UpdateOrCreateCharacterSectionStatus2(ctx, arg.CharacterID, arg.Section, opt)
	if err != nil {
		return false, err
	}
	s.cs.CharacterSet(o)

	slog.Debug("Has section changed", "characterID", arg.CharacterID, "section", arg.Section, "changed", hasChanged)
	return hasChanged, nil
}

func (s *CharacterService) getCharacterSectionStatus(ctx context.Context, characterID int32, section model.CharacterSection) (*model.CharacterSectionStatus, error) {
	o, err := s.st.GetCharacterSectionStatus(ctx, characterID, section)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return o, nil
}

func calcContentHash(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	b2 := md5.Sum(b)
	hash := hex.EncodeToString(b2[:])
	return hash, nil
}
