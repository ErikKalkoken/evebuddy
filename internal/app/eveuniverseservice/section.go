package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func (s *EveUniverseService) getSectionStatus(ctx context.Context, section app.GeneralSection) (*app.GeneralSectionStatus, error) {
	o, err := s.st.GetGeneralSectionStatus(ctx, section)
	if errors.Is(err, app.ErrNotFound) {
		return nil, nil
	}
	return o, err
}

// UpdateSectionIfNeeded updates a section from ESI and returns the IDs of changed objects if there are any.
func (s *EveUniverseService) UpdateSectionIfNeeded(ctx context.Context, arg app.GeneralSectionUpdateParams) (set.Set[int32], error) {
	var zero set.Set[int32]
	if !arg.ForceUpdate {
		status, err := s.getSectionStatus(ctx, arg.Section)
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
	var f func(context.Context) (set.Set[int32], error)
	switch arg.Section {
	case app.SectionEveTypes:
		f = s.updateTypes
	case app.SectionEveCharacters:
		f = s.UpdateAllCharactersESI
	case app.SectionEveCorporations:
		f = s.UpdateAllCorporationsESI
	case app.SectionEveMarketPrices:
		f = s.updateMarketPricesESI
	case app.SectionEveEntities:
		f = s.UpdateAllEntitiesESI
	default:
		slog.Warn("encountered unknown section", "section", arg.Section)
	}
	if arg.OnUpdateStarted != nil && arg.OnUpdateCompleted != nil {
		arg.OnUpdateStarted()
		defer arg.OnUpdateCompleted()
	}
	x, err, _ := s.sfg.Do(fmt.Sprintf("update-general-section-%s", arg.Section), func() (any, error) {
		slog.Debug("Started updating eveuniverse section", "section", arg.Section)
		startedAt := optional.New(time.Now())
		o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, storage.UpdateOrCreateGeneralSectionStatusParams{
			Section:   arg.Section,
			StartedAt: &startedAt,
		})
		if err != nil {
			return false, err
		}
		s.scs.SetGeneralSection(o)
		changed, err := f(ctx)
		slog.Debug("Finished updating general section", "section", arg.Section)
		return changed, err
	})
	if err != nil {
		slog.Error("General section update failed", "section", arg.Section, "error", err)
		errorMessage := app.ErrorDisplay(err)
		startedAt := optional.Optional[time.Time]{}
		o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, storage.UpdateOrCreateGeneralSectionStatusParams{
			Error:     &errorMessage,
			Section:   arg.Section,
			StartedAt: &startedAt,
		})
		if err != nil {
			return zero, err
		}
		s.scs.SetGeneralSection(o)
		return zero, err
	}
	changed := x.(set.Set[int32])
	completedAt := storage.NewNullTimeFromTime(time.Now())
	errorMessage := ""
	startedAt2 := optional.Optional[time.Time]{}
	o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, storage.UpdateOrCreateGeneralSectionStatusParams{
		CompletedAt: &completedAt,
		Error:       &errorMessage,
		Section:     arg.Section,
		StartedAt:   &startedAt2,
	})
	if err != nil {
		return zero, err
	}
	s.scs.SetGeneralSection(o)
	return changed, nil
}
