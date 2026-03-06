package corporationservice

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
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
	hasChanged, err := s.UpdateSectionIfNeeded(
		ctx, app.CorporationSectionUpdateParams{
			CorporationID: corporationID,
			ForceUpdate:   forceUpdate,
			Section:       section,
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
