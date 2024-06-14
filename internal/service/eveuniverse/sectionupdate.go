package eveuniverse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"golang.org/x/sync/errgroup"
)

func (eu *EveUniverseService) getSectionStatus(ctx context.Context, section model.GeneralSection) (*model.GeneralSectionStatus, error) {
	x, err := eu.st.GetGeneralSectionStatus(ctx, section)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	} else if err != nil {
		return x, err
	}
	return x, nil
}

// SectionExists reports whether this section exists at all.
// This allows the app to wait with showing related data to the user until this section is full downloaded for the first time.
func (s *EveUniverseService) SectionExists(section model.GeneralSection) (bool, error) {
	o, err := s.getSectionStatus(context.Background(), section)
	if err != nil {
		return false, err
	}
	if o == nil {
		return false, nil
	}
	return !o.CompletedAt.IsZero(), nil
}

func (s *EveUniverseService) UpdateSection(ctx context.Context, section model.GeneralSection, forceUpdate bool) (bool, error) {
	status, err := s.getSectionStatus(ctx, section)
	if err != nil {
		return false, err
	}
	if status != nil {
		if status.IsOK() && !status.IsExpired() {
			return false, nil
		}
	}

	var f func(context.Context) error
	switch section {
	case model.SectionEveCategories:
		f = s.updateEveCategories
	case model.SectionEveCharacters:
		f = s.UpdateAllEveCharactersESI
	case model.SectionEveMarketPrices:
		f = s.updateEveMarketPricesESI
	}
	key := fmt.Sprintf("Update-section-%s", section)
	_, err, _ = s.sfg.Do(key, func() (any, error) {
		slog.Info("Started updating eveuniverse section", "section", section)
		startedAt := storage.NewNullTime(time.Now())
		arg2 := storage.UpdateOrCreateGeneralSectionStatusParams{
			Section:   section,
			StartedAt: &startedAt,
		}
		o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
		if err != nil {
			return false, err
		}
		s.sc.GeneralSectionSet(o)
		err = f(ctx)
		slog.Info("Finished updating eveuniverse section", "section", section)
		return nil, err
	})
	if err != nil {
		errorMessage := humanize.Error(err)
		startedAt := sql.NullTime{}
		arg2 := storage.UpdateOrCreateGeneralSectionStatusParams{
			Section:   section,
			Error:     &errorMessage,
			StartedAt: &startedAt,
		}
		o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
		if err != nil {
			return false, err
		}
		s.sc.GeneralSectionSet(o)
		return false, err
	}
	completedAt := storage.NewNullTime(time.Now())
	errorMessage := ""
	startedAt2 := sql.NullTime{}
	arg2 := storage.UpdateOrCreateGeneralSectionStatusParams{
		Section: section,

		Error:       &errorMessage,
		CompletedAt: &completedAt,
		StartedAt:   &startedAt2,
	}
	o, err := s.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
	if err != nil {
		return false, err
	}
	s.sc.GeneralSectionSet(o)
	return true, nil
}

func (eu *EveUniverseService) updateEveCategories(ctx context.Context) error {
	g := new(errgroup.Group)
	g.Go(func() error {
		return eu.UpdateEveCategoryWithChildrenESI(ctx, model.EveCategorySkill)
	})
	g.Go(func() error {
		return eu.UpdateEveCategoryWithChildrenESI(ctx, model.EveCategoryShip)
	})
	if err := g.Wait(); err != nil {
		return err
	}
	if err := eu.UpdateEveShipSkills(ctx); err != nil {
		return err
	}
	return nil
}
