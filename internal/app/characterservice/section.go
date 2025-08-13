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
	"github.com/antihax/goesi"
)

// UpdateSectionIfNeeded updates a section from ESI if has expired and changed
// and reports back if it has changed
func (s *CharacterService) UpdateSectionIfNeeded(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.CharacterID == 0 || arg.Section == "" {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	if !arg.ForceUpdate {
		status, err := s.st.GetCharacterSectionStatus(ctx, arg.CharacterID, arg.Section)
		if err != nil {
			if !errors.Is(err, app.ErrNotFound) {
				return false, err
			}
		} else {
			if !status.HasError() && !status.IsExpired() {
				return false, nil
			}
			if status.HasError() && !status.WasUpdatedWithinErrorTimedOut() {
				return false, nil
			}
		}
	}
	var f func(context.Context, app.CharacterUpdateSectionParams) (bool, error)
	switch arg.Section {
	case app.SectionCharacterAssets:
		f = s.updateAssetsESI
	case app.SectionCharacterAttributes:
		f = s.updateAttributesESI
	case app.SectionCharacterContracts:
		f = s.updateContractsESI
	case app.SectionCharacterImplants:
		f = s.updateImplantsESI
	case app.SectionCharacterIndustryJobs:
		f = s.updateIndustryJobsESI
	case app.SectionCharacterJumpClones:
		f = s.updateJumpClonesESI
	case app.SectionCharacterLocation:
		f = s.updateLocationESI
	case app.SectionCharacterMails:
		f = s.updateMailsESI
	case app.SectionCharacterMailLabels:
		f = s.updateMailLabelsESI
	case app.SectionCharacterMailLists:
		f = s.updateMailListsESI
	case app.SectionCharacterNotifications:
		f = s.updateNotificationsESI
	case app.SectionCharacterOnline:
		f = s.updateOnlineESI
	case app.SectionCharacterRoles:
		f = s.updateRolesESI
	case app.SectionCharacterPlanets:
		f = s.updatePlanetsESI
	case app.SectionCharacterShip:
		f = s.updateShipESI
	case app.SectionCharacterSkillqueue:
		f = s.updateSkillqueueESI
	case app.SectionCharacterSkills:
		f = s.updateSkillsESI
	case app.SectionCharacterWalletBalance:
		f = s.updateWalletBalanceESI
	case app.SectionCharacterWalletJournal:
		f = s.updateWalletJournalEntryESI
	case app.SectionCharacterWalletTransactions:
		f = s.updateWalletTransactionESI
	default:
		return false, fmt.Errorf("update section: unknown section: %s", arg.Section)
	}
	key := fmt.Sprintf("update-character-section-%s-%d", arg.Section, arg.CharacterID)
	x, err, _ := s.sfg.Do(key, func() (any, error) {
		arg.OnUpdateStarted()
		defer arg.OnUpdateCompleted()
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
		s.scs.SetCharacterSection(o)
		return false, fmt.Errorf("update character section from ESI for %v: %w", arg, err)
	}
	changed := x.(bool)
	slog.Info("Character section update completed", "characterID", arg.CharacterID, "section", arg.Section, "forced", arg.ForceUpdate, "changed", changed)
	return changed, err
}

// updateSectionIfChanged updates a character section if it has changed
// and reports whether it has changed
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
	s.scs.SetCharacterSection(o)
	token, err := s.GetValidCharacterToken(ctx, arg.CharacterID)
	if err != nil {
		return false, err
	}
	if !token.HasScopes(arg.Section.Scopes()) {
		return false, app.ErrNotFound
	}
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
	data, err := fetch(ctx, arg.CharacterID)
	if err != nil {
		return false, err
	}
	hash, err := calcContentHash(data)
	if err != nil {
		return false, err
	}

	// identify if changed
	var notFound, hasChanged bool
	u, err := s.st.GetCharacterSectionStatus(ctx, arg.CharacterID, arg.Section)
	if errors.Is(err, app.ErrNotFound) {
		notFound = true
	} else if err != nil {
		return false, err
	} else {
		hasChanged = u.ContentHash != hash
	}

	// update if needed
	if arg.ForceUpdate || notFound || hasChanged {
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
	s.scs.SetCharacterSection(o)
	slog.Debug("Has section changed", "characterID", arg.CharacterID, "section", arg.Section, "changed", hasChanged)
	return hasChanged, nil
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
