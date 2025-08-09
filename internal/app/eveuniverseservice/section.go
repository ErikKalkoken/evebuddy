package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func (s *EveUniverseService) getSectionStatus(ctx context.Context, section app.GeneralSection) (*app.GeneralSectionStatus, error) {
	o, err := s.st.GetGeneralSectionStatus(ctx, section)
	if errors.Is(err, app.ErrNotFound) {
		return nil, nil
	}
	return o, err
}

// UpdateSection updates a section from ESI and returns the IDs of changed objects if there are any.
func (s *EveUniverseService) UpdateSection(ctx context.Context, section app.GeneralSection, forceUpdate bool) (set.Set[int32], error) {
	status, err := s.getSectionStatus(ctx, section)
	if err != nil {
		return set.Set[int32]{}, err
	}
	if !forceUpdate && status != nil {
		if !status.HasError() && !status.IsExpired() {
			return set.Set[int32]{}, nil
		}
		if status.HasError() && !status.WasUpdatedWithinErrorTimedOut() {
			return set.Set[int32]{}, nil
		}
	}
	var f func(context.Context) (set.Set[int32], error)
	switch section {
	case app.SectionEveTypes:
		f = func(ctx context.Context) (set.Set[int32], error) {
			err := s.updateCategories(ctx)
			return set.Of[int32](0), err // FIXME: Fake change
		}
	case app.SectionEveCharacters:
		f = s.UpdateAllCharactersESI
	case app.SectionEveCorporations:
		f = s.UpdateAllCorporationsESI
	case app.SectionEveMarketPrices:
		f = func(ctx context.Context) (set.Set[int32], error) {
			err := s.updateMarketPricesESI(ctx)
			return set.Of[int32](0), err // FIXME: Fake change
		}
	case app.SectionEveEntities:
		f = func(ctx context.Context) (set.Set[int32], error) {
			err := s.UpdateAllEntitiesESI(ctx)
			return set.Of[int32](0), err // FIXME: Fake change
		}
	default:
		slog.Warn("encountered unknown section", "section", section)
	}
	x, err, _ := s.sfg.Do(fmt.Sprintf("update-general-section-%s", section), func() (any, error) {
		slog.Debug("Started updating eveuniverse section", "section", section)
		startedAt := optional.New(time.Now())
		arg2 := storage.UpdateOrCreateGeneralSectionStatusParams{
			Section:   section,
			StartedAt: &startedAt,
		}
		o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
		if err != nil {
			return false, err
		}
		s.scs.SetGeneralSection(o)
		changed, err := f(ctx)
		slog.Debug("Finished updating eveuniverse section", "section", section)
		return changed, err
	})
	if err != nil {
		errorMessage := app.ErrorDisplay(err)
		startedAt := optional.Optional[time.Time]{}
		arg2 := storage.UpdateOrCreateGeneralSectionStatusParams{
			Section:   section,
			Error:     &errorMessage,
			StartedAt: &startedAt,
		}
		o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
		if err != nil {
			return set.Set[int32]{}, err
		}
		s.scs.SetGeneralSection(o)
		return set.Set[int32]{}, err
	}
	changed := x.(set.Set[int32])
	completedAt := storage.NewNullTimeFromTime(time.Now())
	errorMessage := ""
	startedAt2 := optional.Optional[time.Time]{}
	arg2 := storage.UpdateOrCreateGeneralSectionStatusParams{
		Section: section,

		Error:       &errorMessage,
		CompletedAt: &completedAt,
		StartedAt:   &startedAt2,
	}
	o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
	if err != nil {
		return set.Set[int32]{}, err
	}
	s.scs.SetGeneralSection(o)
	return changed, nil
}
