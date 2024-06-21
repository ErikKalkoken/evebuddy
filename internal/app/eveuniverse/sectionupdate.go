package eveuniverse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"golang.org/x/sync/errgroup"
)

func (eu *EveUniverseService) getSectionStatus(ctx context.Context, section app.GeneralSection) (*app.GeneralSectionStatus, error) {
	x, err := eu.st.GetGeneralSectionStatus(ctx, section)
	if errors.Is(err, sqlite.ErrNotFound) {
		return nil, nil
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (s *EveUniverseService) UpdateSection(ctx context.Context, section app.GeneralSection, forceUpdate bool) (bool, error) {
	status, err := s.getSectionStatus(ctx, section)
	if err != nil {
		return false, err
	}
	if !forceUpdate && status != nil {
		if status.IsOK() && !status.IsExpired() {
			return false, nil
		}
	}

	var f func(context.Context) error
	switch section {
	case app.SectionEveCategories:
		f = s.updateEveCategories
	case app.SectionEveCharacters:
		f = s.UpdateAllEveCharactersESI
	case app.SectionEveMarketPrices:
		f = s.updateEveMarketPricesESI
	}
	key := fmt.Sprintf("Update-section-%s", section)
	_, err, _ = s.sfg.Do(key, func() (any, error) {
		slog.Info("Started updating eveuniverse section", "section", section)
		startedAt := optional.New(time.Now())
		arg2 := sqlite.UpdateOrCreateGeneralSectionStatusParams{
			Section:   section,
			StartedAt: &startedAt,
		}
		o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
		if err != nil {
			return false, err
		}
		s.StatusCacheService.GeneralSectionSet(o)
		err = f(ctx)
		slog.Info("Finished updating eveuniverse section", "section", section)
		return nil, err
	})
	if err != nil {
		errorMessage := humanize.Error(err)
		startedAt := optional.Optional[time.Time]{}
		arg2 := sqlite.UpdateOrCreateGeneralSectionStatusParams{
			Section:   section,
			Error:     &errorMessage,
			StartedAt: &startedAt,
		}
		o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
		if err != nil {
			return false, err
		}
		s.StatusCacheService.GeneralSectionSet(o)
		return false, err
	}
	completedAt := sqlite.NewNullTime(time.Now())
	errorMessage := ""
	startedAt2 := optional.Optional[time.Time]{}
	arg2 := sqlite.UpdateOrCreateGeneralSectionStatusParams{
		Section: section,

		Error:       &errorMessage,
		CompletedAt: &completedAt,
		StartedAt:   &startedAt2,
	}
	o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
	if err != nil {
		return false, err
	}
	s.StatusCacheService.GeneralSectionSet(o)
	return true, nil
}

func (eu *EveUniverseService) updateEveCategories(ctx context.Context) error {
	g := new(errgroup.Group)
	g.Go(func() error {
		return eu.UpdateEveCategoryWithChildrenESI(ctx, app.EveCategorySkill)
	})
	g.Go(func() error {
		return eu.UpdateEveCategoryWithChildrenESI(ctx, app.EveCategoryShip)
	})
	if err := g.Wait(); err != nil {
		return err
	}
	if err := eu.UpdateEveShipSkills(ctx); err != nil {
		return err
	}
	return nil
}
