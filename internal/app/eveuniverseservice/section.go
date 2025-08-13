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
func (s *EveUniverseService) UpdateSection(ctx context.Context, arg app.GeneralUpdateSectionParams) (set.Set[int32], error) {
	status, err := s.getSectionStatus(ctx, arg.Section)
	if err != nil {
		return set.Set[int32]{}, err
	}
	if !arg.ForceUpdate && status != nil {
		if !status.HasError() && !status.IsExpired() {
			return set.Set[int32]{}, nil
		}
		if status.HasError() && !status.WasUpdatedWithinErrorTimedOut() {
			return set.Set[int32]{}, nil
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
		arg.OnUpdateStarted()
		defer arg.OnUpdateCompleted()
		changed, err := f(ctx)
		slog.Debug("Finished updating general section", "section", arg.Section)
		return changed, err
	})
	if err != nil {
		errorMessage := app.ErrorDisplay(err)
		startedAt := optional.Optional[time.Time]{}
		o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, storage.UpdateOrCreateGeneralSectionStatusParams{
			Error:     &errorMessage,
			Section:   arg.Section,
			StartedAt: &startedAt,
		})
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
	o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, storage.UpdateOrCreateGeneralSectionStatusParams{
		CompletedAt: &completedAt,
		Error:       &errorMessage,
		Section:     arg.Section,
		StartedAt:   &startedAt2,
	})
	if err != nil {
		return set.Set[int32]{}, err
	}
	s.scs.SetGeneralSection(o)
	return changed, nil
}
