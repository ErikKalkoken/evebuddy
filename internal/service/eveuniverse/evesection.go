package eveuniverse

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"golang.org/x/sync/errgroup"
)

// SectionExists reports whether this section exists at all.
// This allows the app to wait with showing related data to the user until this section is full downloaded for the first time.
func (eu *EveUniverseService) SectionExists(section model.EveUniverseSection) (bool, error) {
	_, ok, err := eu.dt.Time(section.KeyCompletedAt())
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (eu *EveUniverseService) UpdateSection(ctx context.Context, section model.EveUniverseSection, forceUpdate bool) (bool, error) {
	lastUpdated, ok, err := eu.dt.Time(section.KeyCompletedAt())
	if err != nil {
		return false, err
	}
	if ok && time.Now().Before(lastUpdated.Add(section.Timeout())) {
		return false, nil
	}

	var f func(context.Context) error
	switch section {
	case model.SectionEveCategories:
		f = eu.updateEveCategories
	case model.SectionEveCharacters:
		f = eu.UpdateAllEveCharactersESI
	case model.SectionEveMarketPrices:
		f = eu.updateEveMarketPricesESI
	}
	key := fmt.Sprintf("Update-section-%s", section)
	_, err, _ = eu.sfg.Do(key, func() (any, error) {
		slog.Info("Started updating eveuniverse section", "section", section)
		if err := eu.dt.SetTime(section.KeyStartedAt(), time.Now()); err != nil {
			return nil, err
		}
		err := f(ctx)
		if err := eu.dt.Delete(section.KeyStartedAt()); err != nil {
			return nil, err
		}
		slog.Info("Finished updating eveuniverse section", "section", section)
		return nil, err
	})
	if err != nil {
		errorMessage := humanize.Error(err)
		if err2 := eu.dt.SetString(section.KeyError(), errorMessage); err2 != nil {
			slog.Error("failed to record error for failed eveuniverse section update: %s", err2)
		}
		return false, err
	}
	if err := eu.dt.SetTime(section.KeyCompletedAt(), time.Now()); err != nil {
		return false, err
	}
	if err := eu.dt.Delete(section.KeyError()); err != nil {
		slog.Error("failed to clear error for eveuniverse section update: %s", err)
	}
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
