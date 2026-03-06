package corporationservice

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xsingleflight"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func (s *CorporationService) StartUpdateTickerCorporations(d time.Duration) {
	go func() {
		for {
			go func() {
				if err := s.UpdateCorporationsIfNeeded(context.Background(), false); err != nil {
					slog.Error("Failed to update corporations", "error", err)
				}
			}()
			<-time.Tick(d)
		}
	}()
}

func (s *CorporationService) UpdateCorporationsIfNeeded(ctx context.Context, forceUpdate bool) error {
	if !forceUpdate && xgoesi.IsDailyDowntime() {
		slog.Info("Skipping regular update of corporations during daily downtime")
		return nil
	}

	id := "corporations-" + s.signals.PseudoUniqueID()
	s.signals.UpdateStarted.Emit(ctx, id)
	defer s.signals.UpdateStopped.Emit(ctx, id)

	changed, err := s.UpdateCorporations(ctx)
	if err != nil {
		return err
	}
	if changed {
		s.signals.CorporationsChanged.Emit(ctx, struct{}{})
	}
	corporations, err := s.ListCorporationIDs(ctx)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	for id := range corporations.All() {
		wg.Go(func() {
			s.UpdateCorporationAndRefreshIfNeeded(ctx, id, forceUpdate)
		})
	}
	slog.Debug("Started updating corporations", "corporations", corporations, "forceUpdate", forceUpdate)
	wg.Wait()
	slog.Debug("Finished updating corporations", "corporations", corporations, "forceUpdate", forceUpdate)
	return nil
}

// UpdateCorporationAndRefreshIfNeeded runs update for all sections of a corporation if needed
// and refreshes the UI accordingly.
func (s *CorporationService) UpdateCorporationAndRefreshIfNeeded(ctx context.Context, corporationID int64, forceUpdate bool) {
	sections := app.CorporationSections
	var wg sync.WaitGroup
	for _, section := range sections {
		wg.Go(func() {
			s.UpdateSectionAndRefreshIfNeeded(ctx, corporationID, section, forceUpdate)
		})
	}
	slog.Debug("Started updating corporation", "corporationID", corporationID, "sections", sections, "forceUpdate", forceUpdate)
	wg.Wait()
	slog.Debug("Finished updating corporation", "corporationID", corporationID, "sections", sections, "forceUpdate", forceUpdate)
}

// UpdateSectionAndRefreshIfNeeded runs update for a corporation section if needed
// and refreshes the UI accordingly.
//
// All UI areas showing data based on corporation sections needs to be included
// to make sure they are refreshed when data changes.
func (s *CorporationService) UpdateSectionAndRefreshIfNeeded(ctx context.Context, corporationID int64, section app.CorporationSection, forceUpdate bool) {
	hasChanged, err := s.updateSectionIfNeeded(
		ctx, corporationSectionUpdateParams{
			corporationID: corporationID,
			forceUpdate:   forceUpdate,
			section:       section,
		},
	)
	if err != nil {
		slog.Error("Failed to update corporation section", "corporationID", corporationID, "section", section, "err", err)
		return
	}
	needsRefresh := hasChanged || forceUpdate
	arg := app.CorporationSectionUpdated{
		CorporationID: corporationID,
		Section:       section,
		NeedsRefresh:  needsRefresh,
	}
	var wg sync.WaitGroup
	if needsRefresh {
		wg.Go(func() {
			s.signals.CorporationSectionChanged.Emit(ctx, arg)
		})
	}
	wg.Go(func() {
		s.signals.CorporationSectionUpdated.Emit(ctx, arg)
	})
	wg.Wait()
}

// RemoveSectionDataWhenPermissionLost removes all data related to a corporation section after the permission was lost.
// This can happen after a character has lost a role or a character was deleted.
func (s *CorporationService) RemoveSectionDataWhenPermissionLost(ctx context.Context, corporationID int64) error {
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
func (s *CorporationService) PermittedSections(ctx context.Context, corporationID int64) (set.Set[app.CorporationSection], error) {
	var enabled, zero set.Set[app.CorporationSection]
	wrapErr := func(err error) error {
		return fmt.Errorf("PermittedSections %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return zero, nil
	}
	for _, section := range app.CorporationSections {
		ok, err := s.hasToken(ctx, corporationID, section.Roles(), section.Scopes())
		if err != nil {
			if errors.Is(err, app.ErrNotFound) {
				continue
			}
			return zero, wrapErr(err)
		}
		if !ok {
			continue
		}
		enabled.Add(section)
	}
	return enabled, nil
}

// PermittedSection reports whether the user has permission to access a section.
func (s *CorporationService) PermittedSection(ctx context.Context, corporationID int64, section app.CorporationSection) (bool, error) {
	sections, err := s.PermittedSections(ctx, corporationID)
	if err != nil {
		return false, err
	}
	return sections.Contains(section), nil
}

type corporationSectionUpdateParams struct {
	corporationID int64
	forceUpdate   bool
	section       app.CorporationSection
}

// updateSectionIfNeeded updates a section from ESI if has expired and changed
// and reports back if it has changed
func (s *CorporationService) updateSectionIfNeeded(ctx context.Context, arg corporationSectionUpdateParams) (bool, error) {
	if arg.corporationID == 0 || arg.section == "" {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	if !arg.forceUpdate {
		status, err := s.st.GetCorporationSectionStatus(ctx, arg.corporationID, arg.section)
		if err != nil {
			if !errors.Is(err, app.ErrNotFound) {
				return false, err
			}
		} else {
			enabled, err := s.PermittedSection(ctx, arg.corporationID, arg.section)
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
	var f func(context.Context, corporationSectionUpdateParams) (bool, error)
	switch arg.section {
	case app.SectionCorporationAssets:
		f = s.updateAssetsESI
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
		return false, fmt.Errorf("update section: unknown section: %s", arg.section)
	}
	key := fmt.Sprintf("update-corporation-section-%s-%d", arg.section, arg.corporationID)
	hasChanged, err, _ := xsingleflight.Do(&s.sfg, key, func() (bool, error) {
		return f(ctx, arg)
	})
	if err != nil {
		slog.Error("Corporation section update failed", "corporationID", arg.corporationID, "section", arg.section, "error", err)
		errorMessage := err.Error()
		startedAt := optional.Optional[time.Time]{}
		o, err2 := s.st.UpdateOrCreateCorporationSectionStatus(ctx, storage.UpdateOrCreateCorporationSectionStatusParams{
			CorporationID: arg.corporationID,
			ErrorMessage:  &errorMessage,
			Section:       arg.section,
			StartedAt:     &startedAt,
		})
		if err2 != nil {
			slog.Error("record error for failed section update", "error", err2)
		}
		s.scs.SetCorporationSection(o)
		return false, fmt.Errorf("update corporation section from ESI for %+v: %w", arg, err)
	}
	slog.Info(
		"Corporation section update completed",
		"corporationID", arg.corporationID,
		"section", arg.section,
		"forced", arg.forceUpdate,
		"hasChanged", hasChanged,
	)
	return hasChanged, err
}

// updateSectionIfChanged updates a character section if it has changed
// and reports whether it has changed
func (s *CorporationService) updateSectionIfChanged(
	ctx context.Context,
	arg corporationSectionUpdateParams,
	skipChangeDetection bool,
	fetch func(ctx context.Context, arg corporationSectionUpdateParams) (any, error), // returns data from ESI
	update func(ctx context.Context, arg corporationSectionUpdateParams, data any) (bool, error), // reports whether it has changed
) (bool, error) {
	startedAt := optional.New(time.Now())
	o, err := s.st.UpdateOrCreateCorporationSectionStatus(ctx, storage.UpdateOrCreateCorporationSectionStatusParams{
		CorporationID: arg.corporationID,
		Section:       arg.section,
		StartedAt:     &startedAt,
	})
	if err != nil {
		return false, err
	}
	s.scs.SetCorporationSection(o)
	var hash, comment string
	var hasChanged bool
	ts, characterID, err := s.cs.TokenSourceForCorporation(ctx, arg.corporationID, arg.section.Roles(), arg.section.Scopes())
	if errors.Is(err, app.ErrNotFound) {
		comment = fmt.Sprintf(
			"update skipped due to missing corporation member with required roles %s and/or missing or invalid token",
			arg.section.Roles(),
		)
		slog.Info(
			"Section "+comment,
			"corporationID", arg.corporationID,
			"section", arg.section,
			"role", arg.section.Roles(),
			"scopes", arg.section.Scopes(),
		)
	} else if err != nil {
		return false, err
	} else {
		slog.Debug("Found valid token for updating corporation section", "corporationID", arg.corporationID, "section", arg.section, "characterID", characterID)
		ctx = xgoesi.NewContextWithAuth(ctx, characterID, ts)
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
		var needsUpdate bool
		if arg.forceUpdate || skipChangeDetection {
			needsUpdate = true
		} else {
			hasChanged, err := s.hasSectionChanged(ctx, arg, hash)
			if err != nil {
				return false, err
			}
			needsUpdate = hasChanged
		}

		if needsUpdate {
			b, err := update(ctx, arg, data)
			if err != nil {
				return false, err
			}
			hasChanged = b
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
		CorporationID: arg.corporationID,
		ErrorMessage:  &errorMessage,
		Section:       arg.section,
		StartedAt:     &startedAt2,
	})
	if err != nil {
		return false, err
	}
	s.scs.SetCorporationSection(o)
	slog.Debug(
		"Has section changed",
		slog.Any("corporationID", arg.corporationID),
		slog.Any("section", arg.section),
		slog.Any("hasChanged", hasChanged),
	)
	return hasChanged, nil
}

func (s *CorporationService) hasSectionChanged(ctx context.Context, arg corporationSectionUpdateParams, hash string) (bool, error) {
	status, err := s.st.GetCorporationSectionStatus(ctx, arg.corporationID, arg.section)
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

func (s *CorporationService) hasToken(ctx context.Context, corporationID int64, roles set.Set[app.Role], scopes set.Set[string]) (bool, error) {
	tokens, err := s.st.ListCharacterTokenForCorporation(ctx, corporationID, roles, scopes)
	if err != nil {
		return false, err
	}
	_, ok := xslices.Pop(&tokens)
	return ok, nil
}
