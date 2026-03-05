package eveuniverseservice

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

const (
	sectionsUpdateTicker = 300 * time.Second
)

func (s *EveUniverseService) StartUpdateTicker() {
	go func() {
		for {
			go s.UpdateSectionsIfNeeded(context.Background(), false)
			<-time.Tick(sectionsUpdateTicker)
		}
	}()
}

func (s *EveUniverseService) UpdateSectionsIfNeeded(ctx context.Context, forceUpdate bool) {
	if !forceUpdate && xgoesi.IsDailyDowntime() {
		slog.Info("Skipping regular update of general sections during daily downtime")
		return
	}

	id := "general-" + s.signals.PseudoUniqueID()
	s.signals.UpdateStarted.Emit(ctx, id)
	defer s.signals.UpdateStopped.Emit(ctx, id)

	sections := set.Of(app.EveUniverseSections...)
	var wg sync.WaitGroup
	for section := range sections.All() {
		wg.Go(func() {
			s.UpdateSectionAndRefreshIfNeeded(ctx, section, forceUpdate)
		})
	}
	slog.Debug("Started updating general sections", "sections", sections, "forceUpdate", forceUpdate)
	wg.Wait()
	slog.Debug("Finished updating general sections", "sections", sections, "forceUpdate", forceUpdate)
}

func (s *EveUniverseService) UpdateSectionAndRefreshIfNeeded(ctx context.Context, section app.EveUniverseSection, forceUpdate bool) {
	logErr := func(err error) {
		slog.Error("Failed to update general section", "section", section, "err", err)
	}
	changedIDs, err := s.UpdateSectionIfNeeded(ctx, app.EveUniverseSectionUpdateParams{
		Section:     section,
		ForceUpdate: forceUpdate,
	})
	if err != nil {
		logErr(err)
		return
	}

	needsRefresh := changedIDs.Size() > 0 || forceUpdate
	arg := app.EveUniverseSectionUpdated{
		Section:      section,
		Changed:      changedIDs,
		NeedsRefresh: needsRefresh,
	}

	var wg sync.WaitGroup
	if needsRefresh {
		wg.Go(func() {
			s.signals.EveUniverseSectionChanged.Emit(ctx, arg)
		})
	}
	wg.Go(func() {
		s.signals.EveUniverseSectionUpdated.Emit(ctx, arg)
	})
	wg.Wait()
}
