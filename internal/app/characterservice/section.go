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
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
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
	var f func(context.Context, app.CharacterSectionUpdateParams) (bool, error)
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
	case app.SectionCharacterMailHeaders:
		f = s.updateMailHeadersESI
	case app.SectionCharacterMailLabels:
		f = s.updateMailLabelsESI
	case app.SectionCharacterMailLists:
		f = s.updateMailListsESI
	case app.SectionCharacterMarketOrders:
		f = s.updateMarketOrdersESI
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
	if arg.OnUpdateStarted != nil && arg.OnUpdateCompleted != nil {
		arg.OnUpdateStarted()
		defer arg.OnUpdateCompleted()
	}
	key := fmt.Sprintf("update-character-section-%s-%d", arg.Section, arg.CharacterID)
	x, err, _ := s.sfg.Do(key, func() (any, error) {
		return f(ctx, arg)
	})
	if err != nil {
		s.recordUpdateFailed(ctx, arg, err)
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
	err := s.recordUpdateStarted(ctx, arg)
	if err != nil {
		return false, err
	}
	token, err := s.GetValidCharacterTokenWithScopes(ctx, arg.CharacterID, arg.Section.Scopes())
	if err != nil {
		return false, err
	}
	ctx = xgoesi.NewContextWithAuth(ctx, token.CharacterID, token.AccessToken)
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
	if err := s.recordUpdateSuccessful(ctx, arg, hash); err != nil {
		return false, err
	}
	slog.Debug(
		"Has section changed",
		"characterID", arg.CharacterID,
		"section", arg.Section,
		"needsUpdate", needsUpdate,
	)
	return needsUpdate, nil
}

func (s *CharacterService) recordUpdateStarted(ctx context.Context, arg app.CharacterSectionUpdateParams) error {
	startedAt := optional.New(time.Now())
	o, err := s.st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID: arg.CharacterID,
		Section:     arg.Section,
		StartedAt:   &startedAt,
	})
	if err != nil {
		return err
	}
	s.scs.SetCharacterSection(o)
	return nil
}

func (s *CharacterService) recordUpdateSuccessful(ctx context.Context, arg app.CharacterSectionUpdateParams, hash string) error {
	completedAt := storage.NewNullTimeFromTime(time.Now())
	errorMessage := ""
	startedAt2 := optional.Optional[time.Time]{}
	o, err := s.st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID:  arg.CharacterID,
		CompletedAt:  &completedAt,
		ContentHash:  &hash,
		ErrorMessage: &errorMessage,
		Section:      arg.Section,
		StartedAt:    &startedAt2,
	})
	if err != nil {
		return err
	}
	s.scs.SetCharacterSection(o)
	return nil
}

func (s *CharacterService) recordUpdateFailed(ctx context.Context, arg app.CharacterSectionUpdateParams, err error) {
	errorMessage := err.Error()
	startedAt := optional.Optional[time.Time]{}
	o, err2 := s.st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID:  arg.CharacterID,
		Section:      arg.Section,
		ErrorMessage: &errorMessage,
		StartedAt:    &startedAt,
	})
	if err2 != nil {
		slog.Error("record error for failed section update: %s", "characterID", arg.CharacterID, "error", err2)
	} else {
		s.scs.SetCharacterSection(o)
	}
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
