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
func (s *CharacterService) UpdateSectionIfNeeded(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
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
	var updateFunc func(context.Context, app.CharacterSectionUpdateParams) (bool, error)
	switch arg.Section {
	case app.SectionCharacterAssets:
		updateFunc = s.updateAssetsESI
	case app.SectionCharacterAttributes:
		updateFunc = s.updateAttributesESI
	case app.SectionCharacterContracts:
		updateFunc = s.updateContractsESI
	case app.SectionCharacterImplants:
		updateFunc = s.updateImplantsESI
	case app.SectionCharacterIndustryJobs:
		updateFunc = s.updateIndustryJobsESI
	case app.SectionCharacterJumpClones:
		updateFunc = s.updateJumpClonesESI
	case app.SectionCharacterLocation:
		updateFunc = s.updateLocationESI
	case app.SectionCharacterMails:
		updateFunc = s.updateMailsESI
	case app.SectionCharacterMarketOrders:
		updateFunc = s.updateMarketOrdersESI
	case app.SectionCharacterMailLabels:
		updateFunc = s.updateMailLabelsESI
	case app.SectionCharacterMailLists:
		updateFunc = s.updateMailListsESI
	case app.SectionCharacterNotifications:
		updateFunc = s.updateNotificationsESI
	case app.SectionCharacterOnline:
		updateFunc = s.updateOnlineESI
	case app.SectionCharacterRoles:
		updateFunc = s.updateRolesESI
	case app.SectionCharacterPlanets:
		updateFunc = s.updatePlanetsESI
	case app.SectionCharacterShip:
		updateFunc = s.updateShipESI
	case app.SectionCharacterSkillqueue:
		updateFunc = s.updateSkillqueueESI
	case app.SectionCharacterSkills:
		updateFunc = s.updateSkillsESI
	case app.SectionCharacterWalletBalance:
		updateFunc = s.updateWalletBalanceESI
	case app.SectionCharacterWalletJournal:
		updateFunc = s.updateWalletJournalEntryESI
	case app.SectionCharacterWalletTransactions:
		updateFunc = s.updateWalletTransactionESI
	default:
		return false, fmt.Errorf("update section: unknown section: %s", arg.Section)
	}
	if arg.OnUpdateStarted != nil && arg.OnUpdateCompleted != nil {
		arg.OnUpdateStarted()
		defer arg.OnUpdateCompleted()
	}
	key := fmt.Sprintf("update-character-section-%s-%d", arg.Section, arg.CharacterID)
	x, err, _ := s.sfg.Do(key, func() (any, error) {
		return updateFunc(ctx, arg)
	})
	if err != nil {
		errorMessage := err.Error()
		startedAt := optional.Optional[time.Time]{}
		o, err2 := s.st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID:  arg.CharacterID,
			Section:      arg.Section,
			ErrorMessage: &errorMessage,
			StartedAt:    &startedAt,
		})
		if err2 != nil {
			slog.Error("record error for failed section update: %s", "error", err2)
		}
		s.scs.SetCharacterSection(o)
		return false, fmt.Errorf("update character section from ESI for %+v: %w", arg, err)
	}
	hasChanged := x.(bool)
	slog.Info(
		"Character section update completed",
		"characterID", arg.CharacterID,
		"section", arg.Section,
		"forced", arg.ForceUpdate,
		"hasChanged", hasChanged,
	)
	return hasChanged, err
}

// updateSectionIfChanged updates a character section if it has changed
// and reports whether it has changed
func (s *CharacterService) updateSectionIfChanged(
	ctx context.Context,
	arg app.CharacterSectionUpdateParams,
	fetch func(ctx context.Context, characterID int32) (any, error),
	update func(ctx context.Context, characterID int32, data any) error,
) (bool, error) {
	startedAt := optional.New(time.Now())
	o, err := s.st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID: arg.CharacterID,
		Section:     arg.Section,
		StartedAt:   &startedAt,
	})
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

	// identify whether update is needed
	var needsUpdate bool
	if arg.ForceUpdate {
		needsUpdate = true
	} else if arg.Section.IsSkippingChangeDetection() {
		needsUpdate = true
	} else {
		hasChanged, err := s.hasSectionChanged(ctx, arg, hash)
		if err != nil {
			return false, err
		}
		needsUpdate = hasChanged
	}

	if needsUpdate {
		if err := update(ctx, arg.CharacterID, data); err != nil {
			return false, err
		}
	}

	// record successful completion
	completedAt := storage.NewNullTimeFromTime(time.Now())
	errorMessage := ""
	startedAt2 := optional.Optional[time.Time]{}
	o, err = s.st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID:  arg.CharacterID,
		CompletedAt:  &completedAt,
		ContentHash:  &hash,
		ErrorMessage: &errorMessage,
		Section:      arg.Section,
		StartedAt:    &startedAt2,
	})
	if err != nil {
		return false, err
	}
	s.scs.SetCharacterSection(o)
	slog.Debug(
		"Has section changed",
		"characterID", arg.CharacterID,
		"section", arg.Section,
		"needsUpdate", needsUpdate,
	)
	return needsUpdate, nil
}

func (s *CharacterService) hasSectionChanged(ctx context.Context, arg app.CharacterSectionUpdateParams, hash string) (bool, error) {
	status, err := s.st.GetCharacterSectionStatus(ctx, arg.CharacterID, arg.Section)
	if errors.Is(err, app.ErrNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	hasChanged := status.ContentHash != hash
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
