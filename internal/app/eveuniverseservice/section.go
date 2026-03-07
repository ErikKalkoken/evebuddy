package eveuniverseservice

import (
	"context"
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
)

func (s *EVEUniverseService) StartUpdateTicker(d time.Duration) {
	go func() {
		for {
			go s.UpdateSectionsIfNeeded(context.Background(), false)
			<-time.Tick(d)
		}
	}()
}

func (s *EVEUniverseService) UpdateSectionsIfNeeded(ctx context.Context, forceUpdate bool) {
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

func (s *EVEUniverseService) UpdateSectionAndRefreshIfNeeded(ctx context.Context, section app.EveUniverseSection, forceUpdate bool) {
	logErr := func(err error) {
		slog.Error("Failed to update general section", "section", section, "err", err)
	}
	changedIDs, err := s.updateSectionIfNeeded(ctx, eveUniverseSectionUpdateParams{
		section:     section,
		forceUpdate: forceUpdate,
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

// HasSection reports whether a section exists at all.
func (s *EVEUniverseService) HasSection(ctx context.Context, section app.EveUniverseSection) (bool, error) {
	x, err := s.st.GetGeneralSectionStatus(ctx, section)
	if errors.Is(err, app.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !x.IsMissing(), nil
}

type eveUniverseSectionUpdateParams struct {
	forceUpdate bool
	section     app.EveUniverseSection
}

// updateSectionIfNeeded updates a section from ESI and returns the IDs of changed objects if there are any.
func (s *EVEUniverseService) updateSectionIfNeeded(ctx context.Context, arg eveUniverseSectionUpdateParams) (set.Set[int64], error) {
	var zero set.Set[int64]
	if !arg.forceUpdate {
		status, err := s.st.GetGeneralSectionStatus(ctx, arg.section)
		if err != nil {
			if !errors.Is(err, app.ErrNotFound) {
				return zero, err
			}
		} else {
			if !status.HasError() && !status.IsExpired() {
				return zero, nil
			}
			if status.HasError() && !status.WasUpdatedWithinErrorTimedOut() {
				return zero, nil
			}
		}
	}
	var f func(context.Context) (set.Set[int64], error)
	switch arg.section {
	case app.SectionEveTypes:
		f = s.updateTypes
	case app.SectionEveCharacters:
		f = s.UpdateAllCharactersESI
	case app.SectionEveCorporations:
		f = s.UpdateAllCorporationsESI
	case app.SectionEveMarketPrices:
		f = s.UpdateMarketPricesESI
	case app.SectionEveEntities:
		f = s.UpdateAllEntitiesESI
	default:
		slog.Warn("encountered unknown section", "section", arg.section)
	}
	changed, err, _ := xsingleflight.Do(&s.sfg, fmt.Sprintf("update-general-section-%s", arg.section), func() (set.Set[int64], error) {
		slog.Debug("Started updating eveuniverse section", "section", arg.section)
		startedAt := optional.New(time.Now())
		o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, storage.UpdateOrCreateGeneralSectionStatusParams{
			Section:   arg.section,
			StartedAt: &startedAt,
		})
		if err != nil {
			return set.Set[int64]{}, err
		}
		s.scs.SetEveUniverseSection(o)
		changed, err := f(ctx)
		slog.Debug("Finished updating general section", "section", arg.section)
		return changed, err
	})
	if err != nil {
		slog.Error("General section update failed", "section", arg.section, "error", err)
		errorMessage := app.ErrorDisplay(err)
		startedAt := optional.Optional[time.Time]{}
		o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, storage.UpdateOrCreateGeneralSectionStatusParams{
			Error:     &errorMessage,
			Section:   arg.section,
			StartedAt: &startedAt,
		})
		if err != nil {
			return zero, err
		}
		s.scs.SetEveUniverseSection(o)
		return zero, err
	}
	completedAt := storage.NewNullTimeFromTime(time.Now())
	errorMessage := ""
	startedAt2 := optional.Optional[time.Time]{}
	o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, storage.UpdateOrCreateGeneralSectionStatusParams{
		CompletedAt: &completedAt,
		Error:       &errorMessage,
		Section:     arg.section,
		StartedAt:   &startedAt2,
	})
	if err != nil {
		return zero, err
	}
	s.scs.SetEveUniverseSection(o)
	return changed, nil
}
