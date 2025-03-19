package eveuniverse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

func (eu *EveUniverseService) GetType(ctx context.Context, id int32) (*app.EveType, error) {
	x, err := eu.st.GetEveType(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverseService) GetOrCreateEveCategoryESI(ctx context.Context, id int32) (*app.EveCategory, error) {
	x, err := eu.st.GetEveCategory(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveCategoryFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverseService) createEveCategoryFromESI(ctx context.Context, id int32) (*app.EveCategory, error) {
	key := fmt.Sprintf("createEveCategoryFromESI-%d", id)
	y, err, _ := eu.sfg.Do(key, func() (any, error) {
		r, _, err := eu.esiClient.ESI.UniverseApi.GetUniverseCategoriesCategoryId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveCategoryParams{
			ID:          id,
			Name:        r.Name,
			IsPublished: r.Published,
		}
		return eu.st.CreateEveCategory(ctx, arg)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveCategory), nil
}

func (eu *EveUniverseService) GetOrCreateEveGroupESI(ctx context.Context, id int32) (*app.EveGroup, error) {
	x, err := eu.st.GetEveGroup(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveGroupFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverseService) createEveGroupFromESI(ctx context.Context, id int32) (*app.EveGroup, error) {
	key := fmt.Sprintf("createEveGroupFromESI-%d", id)
	y, err, _ := eu.sfg.Do(key, func() (any, error) {
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
		if err := eu.st.CreateEveGroup(ctx, arg); err != nil {
			return nil, err
		}
		return eu.st.GetEveGroup(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveGroup), nil
}

func (eu *EveUniverseService) GetOrCreateTypeESI(ctx context.Context, id int32) (*app.EveType, error) {
	x, err := eu.st.GetEveType(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveTypeFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverseService) createEveTypeFromESI(ctx context.Context, id int32) (*app.EveType, error) {
	key := fmt.Sprintf("createEveTypeFromESI-%d", id)
	x, err, _ := eu.sfg.Do(key, func() (any, error) {
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
		if err := eu.st.CreateEveType(ctx, arg); err != nil {
			return nil, err
		}
		for _, o := range t.DogmaAttributes {
			x, err := eu.GetOrCreateEveDogmaAttributeESI(ctx, o.AttributeId)
			if err != nil {
				return nil, err
			}
			switch x.Unit {
			case app.EveUnitGroupID:
				go func(ctx context.Context, groupID int32) {
					_, err := eu.GetOrCreateEveGroupESI(ctx, groupID)
					if err != nil {
						slog.Error("Failed to fetch eve group %d", "ID", groupID, "err", err)
					}
				}(ctx, int32(o.Value))
			case app.EveUnitTypeID:
				go func(ctx context.Context, typeID int32) {
					_, err := eu.GetOrCreateTypeESI(ctx, typeID)
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
			if err := eu.st.CreateEveTypeDogmaAttribute(ctx, arg); err != nil {
				return nil, err
			}
		}
		for _, o := range t.DogmaEffects {
			arg := storage.CreateEveTypeDogmaEffectParams{
				DogmaEffectID: o.EffectId,
				EveTypeID:     id,
				IsDefault:     o.IsDefault,
			}
			if err := eu.st.CreateEveTypeDogmaEffect(ctx, arg); err != nil {
				return nil, err
			}
		}
		return eu.st.GetEveType(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveType), nil
}

func (eu *EveUniverseService) AddMissingEveTypes(ctx context.Context, ids []int32) error {
	missingIDs, err := eu.st.MissingEveTypes(ctx, ids)
	if err != nil {
		return err
	}
	if len(missingIDs) == 0 {
		return nil
	}
	slices.Sort(missingIDs)
	slog.Info("Trying to fetch missing EveTypes from ESI", "count", len(missingIDs))
	for _, id := range missingIDs {
		_, err := eu.GetOrCreateTypeESI(ctx, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (eu *EveUniverseService) UpdateEveCategoryWithChildrenESI(ctx context.Context, categoryID int32) error {
	key := fmt.Sprintf("UpdateEveCategoryWithChildrenESI-%d", categoryID)
	_, err, _ := eu.sfg.Do(key, func() (any, error) {
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
			_, err := eu.GetOrCreateTypeESI(ctx, id)
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

func (eu *EveUniverseService) UpdateEveShipSkills(ctx context.Context) error {
	return eu.st.UpdateEveShipSkills(ctx)
}

func (eu *EveUniverseService) ListTypeDogmaAttributesForType(
	ctx context.Context,
	typeID int32,
) ([]*app.EveTypeDogmaAttribute, error) {
	return eu.st.ListEveTypeDogmaAttributesForType(ctx, typeID)
}
