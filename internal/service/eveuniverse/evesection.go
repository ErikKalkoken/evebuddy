package eveuniverse

import (
	"context"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"golang.org/x/sync/errgroup"
)

// SectionExists reports whether this section exists at all.
// This allows the app to wait with showing related data to the user until this section is full downloaded for the first time.
func (eu *EveUniverseService) SectionExists(s model.EveUniverseSection) (bool, error) {
	_, ok, err := eu.dt.GetTime(s.Key())
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (eu *EveUniverseService) UpdateSection(ctx context.Context, section model.EveUniverseSection, forceUpdate bool) (bool, error) {
	key := section.Key()
	lastUpdated, ok, err := eu.dt.GetTime(key)
	if err != nil {
		return false, err
	}
	timeout := section.Timeout()
	if ok && time.Now().Before(lastUpdated.Add(timeout)) {
		return false, nil
	}

	var f func(context.Context) error
	switch section {
	case model.SectionEveCategories:
		f = eu.updateEveCategories
	case model.SectionEveCharacters:
		f = eu.UpdateAllEveCharactersESI
	}
	slog.Info("Started updating eve universe section", "section", section)
	if err := f(ctx); err != nil {
		return false, err
	}
	slog.Info("Finished updating eve universe section", "section", section)
	if err := eu.dt.SetTime(key, time.Now()); err != nil {
		return false, err
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
