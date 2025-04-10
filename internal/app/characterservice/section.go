package characterservice

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// UpdateSectionIfNeeded updates a section from ESI if has expired and changed
// and reports back if it has changed
func (s *CharacterService) UpdateSectionIfNeeded(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.CharacterID == 0 {
		panic("Invalid character ID")
	}
	if !arg.ForceUpdate {
		status, err := s.getSectionStatus(ctx, arg.CharacterID, arg.Section)
		if err != nil {
			return false, err
		}
		if status != nil {
			if status.IsOK() && !status.IsExpired() {
				return false, nil
			}
		}
	}
	var f func(context.Context, app.CharacterUpdateSectionParams) (bool, error)
	switch arg.Section {
	case app.SectionAssets:
		f = s.updateAssetsESI
	case app.SectionAttributes:
		f = s.updateAttributesESI
	case app.SectionContracts:
		f = s.updateContractsESI
	case app.SectionImplants:
		f = s.updateImplantsESI
	case app.SectionIndustryJobs:
		f = s.updateIndustryJobsESI
	case app.SectionJumpClones:
		f = s.updateJumpClonesESI
	case app.SectionLocation:
		f = s.updateLocationESI
	case app.SectionMails:
		f = s.updateMailsESI
	case app.SectionMailLabels:
		f = s.updateMailLabelsESI
	case app.SectionMailLists:
		f = s.updateMailListsESI
	case app.SectionNotifications:
		f = s.updateNotificationsESI
	case app.SectionOnline:
		f = s.updateOnlineESI
	case app.SectionPlanets:
		f = s.updatePlanetsESI
	case app.SectionShip:
		f = s.updateShipESI
	case app.SectionSkillqueue:
		f = s.UpdateSkillqueueESI
	case app.SectionSkills:
		f = s.updateSkillsESI
	case app.SectionWalletBalance:
		f = s.updateWalletBalanceESI
	case app.SectionWalletJournal:
		f = s.updateWalletJournalEntryESI
	case app.SectionWalletTransactions:
		f = s.updateWalletTransactionESI
	default:
		panic(fmt.Sprintf("Undefined section: %s", arg.Section))
	}
	key := fmt.Sprintf("UpdateESI-%s-%d", arg.Section, arg.CharacterID)
	x, err, _ := s.sfg.Do(key, func() (any, error) {
		return f(ctx, arg)
	})
	if err != nil {
		errorMessage := err.Error()
		startedAt := optional.Optional[time.Time]{}
		arg2 := storage.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID:  arg.CharacterID,
			Section:      arg.Section,
			ErrorMessage: &errorMessage,
			StartedAt:    &startedAt,
		}
		o, err2 := s.st.UpdateOrCreateCharacterSectionStatus(ctx, arg2)
		if err2 != nil {
			slog.Error("record error for failed section update: %s", "error", err2)
		}
		s.StatusCacheService.CharacterSectionSet(o)
		return false, fmt.Errorf("update character section from ESI for %v: %w", arg, err)
	}
	changed := x.(bool)
	return changed, err
}

// updateSectionIfChanged updates a character section if it has changed
// and reports wether it has changed
func (s *CharacterService) updateSectionIfChanged(
	ctx context.Context,
	arg app.CharacterUpdateSectionParams,
	fetch func(ctx context.Context, characterID int32) (any, error),
	update func(ctx context.Context, characterID int32, data any) error,
) (bool, error) {
	startedAt := optional.New(time.Now())
	arg2 := storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID: arg.CharacterID,
		Section:     arg.Section,
		StartedAt:   &startedAt,
	}
	o, err := s.st.UpdateOrCreateCharacterSectionStatus(ctx, arg2)
	if err != nil {
		return false, err
	}
	s.StatusCacheService.CharacterSectionSet(o)
	token, err := s.getValidCharacterToken(ctx, arg.CharacterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
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
	u, err := s.getSectionStatus(ctx, arg.CharacterID, arg.Section)
	if err != nil {
		return false, err
	}
	if u == nil {
		hasChanged = true
	} else {
		hasChanged = u.ContentHash != hash
	}
	// update if needed
	if arg.ForceUpdate || hasChanged {
		if err := update(ctx, arg.CharacterID, data); err != nil {
			return false, err
		}
	}

	// record successful completion
	completedAt := storage.NewNullTimeFromTime(time.Now())
	errorMessage := ""
	startedAt2 := optional.Optional[time.Time]{}
	arg2 = storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID: arg.CharacterID,
		Section:     arg.Section,

		ErrorMessage: &errorMessage,
		ContentHash:  &hash,
		CompletedAt:  &completedAt,
		StartedAt:    &startedAt2,
	}
	o, err = s.st.UpdateOrCreateCharacterSectionStatus(ctx, arg2)
	if err != nil {
		return false, err
	}
	s.StatusCacheService.CharacterSectionSet(o)

	slog.Debug("Has section changed", "characterID", arg.CharacterID, "section", arg.Section, "changed", hasChanged)
	return hasChanged, nil
}

func (s *CharacterService) getSectionStatus(ctx context.Context, characterID int32, section app.CharacterSection) (*app.CharacterSectionStatus, error) {
	o, err := s.st.GetCharacterSectionStatus(ctx, characterID, section)
	if errors.Is(err, app.ErrNotFound) {
		return nil, nil
	}
	return o, err
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
