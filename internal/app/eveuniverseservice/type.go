package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"golang.org/x/sync/errgroup"
)

func eveEntityCategoryFromESICategory(c string) app.EveEntityCategory {
	categoryMap := map[string]app.EveEntityCategory{
		"alliance":       app.EveEntityAlliance,
		"character":      app.EveEntityCharacter,
		"corporation":    app.EveEntityCorporation,
		"constellation":  app.EveEntityConstellation,
		"faction":        app.EveEntityFaction,
		"inventory_type": app.EveEntityInventoryType,
		"mailing_list":   app.EveEntityMailList,
		"region":         app.EveEntityRegion,
		"solar_system":   app.EveEntitySolarSystem,
		"station":        app.EveEntityStation,
	}
	c2, ok := categoryMap[c]
	if !ok {
		return app.EveEntityUnknown
	}
	return c2
}

func (s *EveUniverseService) GetType(ctx context.Context, id int32) (*app.EveType, error) {
	return s.st.GetEveType(ctx, id)
}

func (s *EveUniverseService) GetOrCreateCategoryESI(ctx context.Context, id int32) (*app.EveCategory, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateCategoryESI-%d", id), func() (any, error) {
		o1, err := s.st.GetEveCategory(ctx, id)
		if err == nil {
			return o1, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		r, _, err := s.esiClient.ESI.UniverseApi.GetUniverseCategoriesCategoryId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveCategoryParams{
			ID:          id,
			Name:        r.Name,
			IsPublished: r.Published,
		}
		o2, err := s.st.CreateEveCategory(ctx, arg)
		if err != nil {
			return nil, err
		}
		slog.Info("Created eve category", "ID", id)
		return o2, nil
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveCategory), nil
}

func (s *EveUniverseService) GetOrCreateGroupESI(ctx context.Context, id int32) (*app.EveGroup, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateGroupESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveGroup(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		group, _, err := s.esiClient.ESI.UniverseApi.GetUniverseGroupsGroupId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		c, err := s.GetOrCreateCategoryESI(ctx, group.CategoryId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveGroupParams{
			ID:          id,
			Name:        group.Name,
			CategoryID:  c.ID,
			IsPublished: group.Published,
		}
		if err := s.st.CreateEveGroup(ctx, arg); err != nil {
			return nil, err
		}
		slog.Info("Created eve group", "ID", id)
		return s.st.GetEveGroup(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveGroup), nil
}

func (s *EveUniverseService) GetOrCreateTypeESI(ctx context.Context, id int32) (*app.EveType, error) {
	x, err, _ := s.sfg.Do(fmt.Sprintf("GetOrCreateTypeESI-%d", id), func() (any, error) {
		o, err := s.st.GetEveType(ctx, id)
		if err == nil {
			return o, err
		} else if !errors.Is(err, app.ErrNotFound) {
			return nil, err
		}
		t, _, err := s.esiClient.ESI.UniverseApi.GetUniverseTypesTypeId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		g, err := s.GetOrCreateGroupESI(ctx, t.GroupId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveTypeParams{
			ID:             id,
			GroupID:        g.ID,
			Capacity:       t.Capacity,
			Description:    t.Description,
			GraphicID:      t.GraphicId,
			IconID:         t.IconId,
			IsPublished:    t.Published,
			MarketGroupID:  t.MarketGroupId,
			Mass:           t.Mass,
			Name:           t.Name,
			PackagedVolume: t.PackagedVolume,
			PortionSize:    int(t.PortionSize),
			Radius:         t.Radius,
			Volume:         t.Volume,
		}
		if err := s.st.CreateEveType(ctx, arg); err != nil {
			return nil, err
		}
		for _, o := range t.DogmaAttributes {
			x, err := s.GetOrCreateDogmaAttributeESI(ctx, o.AttributeId)
			if err != nil {
				return nil, err
			}
			switch x.Unit {
			case app.EveUnitGroupID:
				go func(ctx context.Context, groupID int32) {
					_, err := s.GetOrCreateGroupESI(ctx, groupID)
					if err != nil {
						slog.Error("Failed to fetch eve group %d", "ID", groupID, "err", err)
					}
				}(ctx, int32(o.Value))
			case app.EveUnitTypeID:
				go func(ctx context.Context, typeID int32) {
					_, err := s.GetOrCreateTypeESI(ctx, typeID)
					if err != nil {
						slog.Error("Failed to fetch eve type %d", "ID", typeID, "err", err)
					}
				}(ctx, int32(o.Value))
			}
			arg := storage.CreateEveTypeDogmaAttributeParams{
				DogmaAttributeID: o.AttributeId,
				EveTypeID:        id,
				Value:            o.Value,
			}
			if err := s.st.CreateEveTypeDogmaAttribute(ctx, arg); err != nil {
				return nil, err
			}
		}
		for _, o := range t.DogmaEffects {
			arg := storage.CreateEveTypeDogmaEffectParams{
				DogmaEffectID: o.EffectId,
				EveTypeID:     id,
				IsDefault:     o.IsDefault,
			}
			if err := s.st.CreateEveTypeDogmaEffect(ctx, arg); err != nil {
				return nil, err
			}
		}
		slog.Info("Created eve type", "ID", id)
		return s.st.GetEveType(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveType), nil
}

// AddMissingTypes fetches missing typeIDs from ESI.
func (s *EveUniverseService) AddMissingTypes(ctx context.Context, ids set.Set[int32]) error {
	missing, err := s.st.MissingEveTypes(ctx, ids)
	if err != nil {
		return err
	}
	if missing.Size() == 0 {
		return nil
	}
	slog.Debug("Trying to fetch missing EveTypes from ESI", "count", missing.Size())
	g := new(errgroup.Group)
	for id := range missing.All() {
		g.Go(func() error {
			_, err := s.GetOrCreateTypeESI(ctx, id)
			return err
		})
	}
	return g.Wait()
}

func (s *EveUniverseService) UpdateCategoryWithChildrenESI(ctx context.Context, categoryID int32) error {
	_, err, _ := s.sfg.Do(fmt.Sprintf("UpdateCategoryWithChildrenESI-%d", categoryID), func() (any, error) {
		var typeIds set.Set[int32]
		_, err := s.GetOrCreateCategoryESI(ctx, categoryID)
		if err != nil {
			return nil, err
		}
		category, _, err := s.esiClient.ESI.UniverseApi.GetUniverseCategoriesCategoryId(ctx, categoryID, nil)
		if err != nil {
			return nil, err
		}
		g := new(errgroup.Group)
		for _, id := range category.Groups {
			g.Go(func() error {
				_, err := s.GetOrCreateGroupESI(ctx, id)
				return err
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
		groupTypes := make([][]int32, len(category.Groups))
		g = new(errgroup.Group)
		for i, id := range category.Groups {
			g.Go(func() error {
				group, _, err := s.esiClient.ESI.UniverseApi.GetUniverseGroupsGroupId(ctx, id, nil)
				if err != nil {
					return err
				}
				groupTypes[i] = group.Types
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
		for _, ids := range groupTypes {
			typeIds.AddSeq(slices.Values(ids))
		}
		if err := s.AddMissingTypes(ctx, typeIds); err != nil {
			return nil, err
		}
		slog.Info("Updated eve types", "categoryID", categoryID, "count", typeIds.Size())
		return nil, nil
	})
	if err != nil {
		return err
	}
	return nil
}
