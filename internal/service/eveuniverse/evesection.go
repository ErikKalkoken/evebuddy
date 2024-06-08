package eveuniverse

import (
	"context"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
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

func (eu *EveUniverseService) UpdateSection(ctx context.Context, s model.EveUniverseSection, forceUpdate bool) (bool, error) {
	var f func(context.Context) error
	switch s {
	case model.SectionEveCategories:
		f = eu.updateEveCategories
	case model.SectionEveCharacters:
		f = eu.updateEveCharacters
	}
	if err := f(ctx); err != nil {
		slog.Error("Failed to update eveuniverse section", "section", s, "err", err)
	}
	return false, nil
}

func (eu *EveUniverseService) updateEveCharacters(ctx context.Context) error {
	key := model.SectionEveCharacters.Key()
	lastUpdated, ok, err := eu.dt.GetTime(key)
	if err != nil {
		return err
	}
	timeout := model.SectionEveCharacters.Timeout()
	if ok && time.Now().Before(lastUpdated.Add(timeout)) {
		return nil
	}
	slog.Info("Started updating eve characters")
	if err := eu.UpdateAllEveCharactersESI(ctx); err != nil {
		return err
	}
	slog.Info("Finished updating eve characters")
	if err := eu.dt.SetTime(key, time.Now()); err != nil {
		return err
	}
	return nil
}

func (eu *EveUniverseService) updateEveCategories(ctx context.Context) error {
	key := model.SectionEveCategories.Key()
	lastUpdated, ok, err := eu.dt.GetTime(key)
	if err != nil {
		return err
	}
	timeout := model.SectionEveCharacters.Timeout()
	if ok && time.Now().Before(lastUpdated.Add(timeout)) {
		return nil
	}
	slog.Info("Started updating categories")
	if err := eu.UpdateEveCategoryWithChildrenESI(ctx, model.EveCategorySkill); err != nil {
		return err
	}
	if err := eu.UpdateEveCategoryWithChildrenESI(ctx, model.EveCategoryShip); err != nil {
		return err
	}
	if err := eu.UpdateEveShipSkills(ctx); err != nil {
		return err
	}
	slog.Info("Finished updating categories")
	if err := eu.dt.SetTime(key, time.Now()); err != nil {
		return err
	}
	return nil
}
