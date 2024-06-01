package eveuniverse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (eu *EveUniverse) GetEveType(ctx context.Context, id int32) (*model.EveType, error) {
	return eu.s.GetEveType(ctx, id)
}

func (eu *EveUniverse) GetOrCreateEveCategoryESI(ctx context.Context, id int32) (*model.EveCategory, error) {
	x, err := eu.s.GetEveCategory(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveCategoryFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverse) createEveCategoryFromESI(ctx context.Context, id int32) (*model.EveCategory, error) {
	key := fmt.Sprintf("createEveCategoryFromESI-%d", id)
	y, err, _ := eu.singleGroup.Do(key, func() (any, error) {
		r, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseCategoriesCategoryId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveCategoryParams{
			ID:          id,
			Name:        r.Name,
			IsPublished: r.Published,
		}
		return eu.s.CreateEveCategory(ctx, arg)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveCategory), nil
}

func (eu *EveUniverse) GetOrCreateEveGroupESI(ctx context.Context, id int32) (*model.EveGroup, error) {
	x, err := eu.s.GetEveGroup(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveGroupFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverse) createEveGroupFromESI(ctx context.Context, id int32) (*model.EveGroup, error) {
	key := fmt.Sprintf("createEveGroupFromESI-%d", id)
	y, err, _ := eu.singleGroup.Do(key, func() (any, error) {
		r, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseGroupsGroupId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		c, err := eu.GetOrCreateEveCategoryESI(ctx, r.CategoryId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveGroupParams{
			ID:          id,
			Name:        r.Name,
			CategoryID:  c.ID,
			IsPublished: r.Published,
		}
		if err := eu.s.CreateEveGroup(ctx, arg); err != nil {
			return nil, err
		}
		return eu.s.GetEveGroup(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveGroup), nil
}

func (eu *EveUniverse) GetOrCreateEveTypeESI(ctx context.Context, id int32) (*model.EveType, error) {
	x, err := eu.s.GetEveType(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveTypeFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverse) createEveTypeFromESI(ctx context.Context, id int32) (*model.EveType, error) {
	key := fmt.Sprintf("createEveTypeFromESI-%d", id)
	x, err, _ := eu.singleGroup.Do(key, func() (any, error) {
		t, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseTypesTypeId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		g, err := eu.GetOrCreateEveGroupESI(ctx, t.GroupId)
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
		if err := eu.s.CreateEveType(ctx, arg); err != nil {
			return nil, err
		}
		for _, o := range t.DogmaAttributes {
			arg := storage.CreateEveTypeDogmaAttributeParams{
				DogmaAttributeID: o.AttributeId,
				EveTypeID:        id,
				Value:            o.Value,
			}
			if err := eu.s.CreateEveTypeDogmaAttribute(ctx, arg); err != nil {
				return nil, err
			}
		}
		for _, o := range t.DogmaEffects {
			arg := storage.CreateEveTypeDogmaEffectParams{
				DogmaEffectID: o.EffectId,
				EveTypeID:     id,
				IsDefault:     o.IsDefault,
			}
			if err := eu.s.CreateEveTypeDogmaEffect(ctx, arg); err != nil {
				return nil, err
			}
		}
		return eu.s.GetEveType(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*model.EveType), nil
}

func (eu *EveUniverse) AddMissingEveTypes(ctx context.Context, ids []int32) error {
	missingIDs, err := eu.s.MissingEveTypes(ctx, ids)
	if err != nil {
		return err
	}
	if len(missingIDs) == 0 {
		return nil
	}
	slices.Sort(missingIDs)
	slog.Info("Trying to fetch missing EveTypes from ESI", "count", len(missingIDs))
	for _, id := range missingIDs {
		_, err := eu.GetOrCreateEveTypeESI(ctx, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (eu *EveUniverse) UpdateEveCategoryWithChildrenESI(ctx context.Context, categoryID int32) error {
	key := fmt.Sprintf("UpdateEveCategoryWithChildrenESI-%d", categoryID)
	_, err, _ := eu.singleGroup.Do(key, func() (any, error) {
		typeIDs := make([]int32, 0)
		r1, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseCategoriesCategoryId(ctx, categoryID, nil)
		if err != nil {
			return nil, err
		}
		for _, id := range r1.Groups {
			r2, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseGroupsGroupId(ctx, id, nil)
			if err != nil {
				return nil, err
			}
			typeIDs = slices.Concat(typeIDs, r2.Types)
		}
		for _, id := range typeIDs {
			_, err := eu.GetOrCreateEveTypeESI(ctx, id)
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	if err != nil {
		return err
	}
	return nil
}
