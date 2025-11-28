package corporationservice

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
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
)

// RemoveSectionDataWhenPermissionLost removes all data related to a corporation section after the permission was lost.
// This can happen after a character has lost a role or a character was deleted.
func (s *CorporationService) RemoveSectionDataWhenPermissionLost(ctx context.Context, corporationID int32) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("RemoveSectionDataWhenNoPermission: CorporationID %d: %w", corporationID, err)
	}
	permitted, err := s.PermittedSections(ctx, corporationID)
	if err != nil {
		return wrapErr(err)
	}
	for _, section := range app.CorporationSections {
		if !permitted.Contains(section) {
			switch section {
			case app.SectionCorporationWalletBalances:
				if err := s.st.DeleteCorporationWalletBalance(ctx, corporationID); err != nil {
					return wrapErr(err)
				}
			case
				app.SectionCorporationWalletJournal1,
				app.SectionCorporationWalletJournal2,
				app.SectionCorporationWalletJournal3,
				app.SectionCorporationWalletJournal4,
				app.SectionCorporationWalletJournal5,
				app.SectionCorporationWalletJournal6,
				app.SectionCorporationWalletJournal7:
				if err := s.st.DeleteCorporationWalletJournal(ctx, corporationID, section.Division()); err != nil {
					return wrapErr(err)
				}
			case
				app.SectionCorporationWalletTransactions1,
				app.SectionCorporationWalletTransactions2,
				app.SectionCorporationWalletTransactions3,
				app.SectionCorporationWalletTransactions4,
				app.SectionCorporationWalletTransactions5,
				app.SectionCorporationWalletTransactions6,
				app.SectionCorporationWalletTransactions7:
				if err := s.st.DeleteCorporationWalletTransactions(ctx, corporationID, section.Division()); err != nil {
					return wrapErr(err)
				}
			case app.SectionCorporationIndustryJobs:
				if err := s.st.DeleteCorporationIndustryJobs(ctx, corporationID); err != nil {
					return wrapErr(err)
				}
			default:
				continue
			}
			err := s.st.ResetCorporationSectionStatusContentHash(ctx, storage.CorporationSectionParams{
				CorporationID: corporationID,
				Section:       section,
			})
			if err != nil {
				return wrapErr(err)
			}
		}
	}
	return nil
}

// PermittedSections returns which sections the user has permission to access.
// i.e. the user has a character with the required roles and scopes.
func (s *CorporationService) PermittedSections(ctx context.Context, corporationID int32) (set.Set[app.CorporationSection], error) {
	var enabled set.Set[app.CorporationSection]
	wrapErr := func(err error) error {
		return fmt.Errorf("PermittedSections %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return enabled, nil
	}
	for _, section := range app.CorporationSections {
		_, err := s.cs.CharacterTokenForCorporation(ctx, corporationID, section.Roles(), section.Scopes(), false)
		if errors.Is(err, app.ErrNotFound) {
			continue
		}
		if err != nil {
			return enabled, wrapErr(err)
		}
		enabled.Add(section)
	}
	return enabled, nil
}

// PermittedSection reports whether the user has permission to access a section.
func (s *CorporationService) PermittedSection(ctx context.Context, corporationID int32, section app.CorporationSection) (bool, error) {
	sections, err := s.PermittedSections(ctx, corporationID)
	if err != nil {
		return false, err
	}
	return sections.Contains(section), nil
}

// UpdateSectionIfNeeded updates a section from ESI if has expired and changed
// and reports back if it has changed
func (s *CorporationService) UpdateSectionIfNeeded(ctx context.Context, arg app.CorporationSectionUpdateParams) (bool, error) {
	if arg.CorporationID == 0 || arg.Section == "" {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	if !arg.ForceUpdate {
		status, err := s.st.GetCorporationSectionStatus(ctx, arg.CorporationID, arg.Section)
		if err != nil {
			if !errors.Is(err, app.ErrNotFound) {
				return false, err
			}
		} else {
			enabled, err := s.PermittedSection(ctx, arg.CorporationID, arg.Section)
			if err != nil {
				slog.Error("Failed to check enabled sections", "error", err)
				enabled = false
			}
			enabledRole := enabled && !status.HasContent()
			if !enabledRole && !status.HasError() && !status.IsExpired() {
				return false, nil
			}
			if status.HasError() && !status.WasUpdatedWithinErrorTimedOut() {
				return false, nil
			}
		}
	}
	var f func(context.Context, app.CorporationSectionUpdateParams) (bool, error)
	switch arg.Section {
	case app.SectionCorporationContracts:
		f = s.updateContractsESI
	case app.SectionCorporationDivisions:
		f = s.updateDivisionsESI
	case app.SectionCorporationIndustryJobs:
		f = s.updateIndustryJobsESI
	case app.SectionCorporationMembers:
		f = s.updateMembersESI
	case app.SectionCorporationStructures:
		f = s.updateStructuresESI
	case app.SectionCorporationWalletBalances:
		f = s.updateWalletBalancesESI
	case
		app.SectionCorporationWalletJournal1,
		app.SectionCorporationWalletJournal2,
		app.SectionCorporationWalletJournal3,
		app.SectionCorporationWalletJournal4,
		app.SectionCorporationWalletJournal5,
		app.SectionCorporationWalletJournal6,
		app.SectionCorporationWalletJournal7:
		f = s.updateWalletJournalESI
	case
		app.SectionCorporationWalletTransactions1,
		app.SectionCorporationWalletTransactions2,
		app.SectionCorporationWalletTransactions3,
		app.SectionCorporationWalletTransactions4,
		app.SectionCorporationWalletTransactions5,
		app.SectionCorporationWalletTransactions6,
		app.SectionCorporationWalletTransactions7:
		f = s.updateWalletTransactionESI
	default:
		return false, fmt.Errorf("update section: unknown section: %s", arg.Section)
	}
	if arg.OnUpdateStarted != nil && arg.OnUpdateCompleted != nil {
		arg.OnUpdateStarted()
		defer arg.OnUpdateCompleted()
	}
	key := fmt.Sprintf("update-corporation-section-%s-%d", arg.Section, arg.CorporationID)
	x, err, _ := s.sfg.Do(key, func() (any, error) {
		return f(ctx, arg)
	})
	if err != nil {
		errorMessage := err.Error()
		startedAt := optional.Optional[time.Time]{}
		o, err2 := s.st.UpdateOrCreateCorporationSectionStatus(ctx, storage.UpdateOrCreateCorporationSectionStatusParams{
			CorporationID: arg.CorporationID,
			ErrorMessage:  &errorMessage,
			Section:       arg.Section,
			StartedAt:     &startedAt,
		})
		if err2 != nil {
			slog.Error("record error for failed section update: %s", "error", err2)
		}
		s.scs.SetCorporationSection(o)
		return false, fmt.Errorf("update corporation section from ESI for %+v: %w", arg, err)
	}
	hasChanged := x.(bool)
	slog.Info(
		"Corporation section update completed",
		"corporationID", arg.CorporationID,
		"section", arg.Section,
		"forced", arg.ForceUpdate,
		"hasChanged", hasChanged,
	)
	return hasChanged, err
}

// updateSectionIfChanged updates a character section if it has changed
// and reports whether it has changed
func (s *CorporationService) updateSectionIfChanged(
	ctx context.Context,
	arg app.CorporationSectionUpdateParams,
	fetch func(ctx context.Context, arg app.CorporationSectionUpdateParams) (any, error),
	update func(ctx context.Context, arg app.CorporationSectionUpdateParams, data any) error,
) (bool, error) {
	startedAt := optional.New(time.Now())
	o, err := s.st.UpdateOrCreateCorporationSectionStatus(ctx, storage.UpdateOrCreateCorporationSectionStatusParams{
		CorporationID: arg.CorporationID,
		Section:       arg.Section,
		StartedAt:     &startedAt,
	})
	if err != nil {
		return false, err
	}
	s.scs.SetCorporationSection(o)
	var hash, comment string
	var needsUpdate bool
	token, err := s.cs.CharacterTokenForCorporation(ctx, arg.CorporationID, arg.Section.Roles(), arg.Section.Scopes(), true)
	if errors.Is(err, app.ErrNotFound) {
		comment = fmt.Sprintf(
			"update skipped due to missing corporation member with required roles %s and/or missing or invalid token",
			arg.Section.Roles(),
		)
		slog.Info(
			"Section "+comment,
			"corporationID", arg.CorporationID,
			"section", arg.Section,
			"role", arg.Section.Roles(),
			"scopes", arg.Section.Scopes(),
		)
	} else if err != nil {
		return false, err
	} else {
		slog.Info("Found valid token for updating corporation section", "corporationID", arg.CorporationID, "section", arg.Section, "characterID", token.CharacterID)
		ctx = xesi.NewContextWithAuth(ctx, token.CharacterID, token.AccessToken)
		data, err := fetch(ctx, arg)
		if err != nil {
			return false, err
		}
		h, err := calcContentHash(data)
		if err != nil {
			return false, err
		}
		hash = h

		// identify whether update is needed
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
			if err := update(ctx, arg, data); err != nil {
				return false, err
			}
		}
	}

	// record completion
	completedAt := storage.NewNullTimeFromTime(time.Now())
	errorMessage := ""
	startedAt2 := optional.Optional[time.Time]{}
	o, err = s.st.UpdateOrCreateCorporationSectionStatus(ctx, storage.UpdateOrCreateCorporationSectionStatusParams{
		Comment:       &comment,
		CompletedAt:   &completedAt,
		ContentHash:   &hash,
		CorporationID: arg.CorporationID,
		ErrorMessage:  &errorMessage,
		Section:       arg.Section,
		StartedAt:     &startedAt2,
	})
	if err != nil {
		return false, err
	}
	s.scs.SetCorporationSection(o)
	slog.Debug(
		"Has section changed",
		"corporationID", arg.CorporationID,
		"section", arg.Section,
		"needsUpdate", needsUpdate,
	)
	return needsUpdate, nil
}

func (s *CorporationService) hasSectionChanged(ctx context.Context, arg app.CorporationSectionUpdateParams, hash string) (bool, error) {
	status, err := s.st.GetCorporationSectionStatus(ctx, arg.CorporationID, arg.Section)
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
