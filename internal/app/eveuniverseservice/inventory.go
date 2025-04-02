package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

func (s *EveUniverseService) GetType(ctx context.Context, id int32) (*app.EveType, error) {
	return s.st.GetEveType(ctx, id)
}

func (s *EveUniverseService) GetOrCreateCategoryESI(ctx context.Context, id int32) (*app.EveCategory, error) {
	o, err := s.st.GetEveCategory(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return s.createCategoryFromESI(ctx, id)
	}
	return o, err
}

func (s *EveUniverseService) createCategoryFromESI(ctx context.Context, id int32) (*app.EveCategory, error) {
	key := fmt.Sprintf("createCategoryFromESI-%d", id)
	y, err, _ := s.sfg.Do(key, func() (any, error) {
		r, _, err := s.esiClient.ESI.UniverseApi.GetUniverseCategoriesCategoryId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveCategoryParams{
			ID:          id,
			Name:        r.Name,
			IsPublished: r.Published,
		}
		return s.st.CreateEveCategory(ctx, arg)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveCategory), nil
}

func (s *EveUniverseService) GetOrCreateGroupESI(ctx context.Context, id int32) (*app.EveGroup, error) {
	o, err := s.st.GetEveGroup(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return s.createGroupFromESI(ctx, id)
	}
	return o, err
}

func (s *EveUniverseService) createGroupFromESI(ctx context.Context, id int32) (*app.EveGroup, error) {
	key := fmt.Sprintf("createGroupFromESI-%d", id)
	y, err, _ := s.sfg.Do(key, func() (any, error) {
		r, _, err := s.esiClient.ESI.UniverseApi.GetUniverseGroupsGroupId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		c, err := s.GetOrCreateCategoryESI(ctx, r.CategoryId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveGroupParams{
			ID:          id,
			Name:        r.Name,
			CategoryID:  c.ID,
			IsPublished: r.Published,
		}
		if err := s.st.CreateEveGroup(ctx, arg); err != nil {
			return nil, err
		}
		return s.st.GetEveGroup(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveGroup), nil
}

func (s *EveUniverseService) GetOrCreateTypeESI(ctx context.Context, id int32) (*app.EveType, error) {
	o, err := s.st.GetEveType(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return s.createTypeFromESI(ctx, id)
	}
	return o, err
}

func (s *EveUniverseService) createTypeFromESI(ctx context.Context, id int32) (*app.EveType, error) {
	key := fmt.Sprintf("createTypeFromESI-%d", id)
	x, err, _ := s.sfg.Do(key, func() (any, error) {
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
		return s.st.GetEveType(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.EveType), nil
}

func (s *EveUniverseService) AddMissingTypes(ctx context.Context, ids []int32) error {
	missingIDs, err := s.st.MissingEveTypes(ctx, ids)
	if err != nil {
		return err
	}
	if len(missingIDs) == 0 {
		return nil
	}
	slices.Sort(missingIDs)
	slog.Debug("Trying to fetch missing EveTypes from ESI", "count", len(missingIDs))
	for _, id := range missingIDs {
		_, err := s.GetOrCreateTypeESI(ctx, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *EveUniverseService) UpdateCategoryWithChildrenESI(ctx context.Context, categoryID int32) error {
	key := fmt.Sprintf("UpdateCategoryWithChildrenESI-%d", categoryID)
	_, err, _ := s.sfg.Do(key, func() (any, error) {
		typeIDs := make([]int32, 0)
		r1, _, err := s.esiClient.ESI.UniverseApi.GetUniverseCategoriesCategoryId(ctx, categoryID, nil)
		if err != nil {
			return nil, err
		}
		for _, id := range r1.Groups {
			r2, _, err := s.esiClient.ESI.UniverseApi.GetUniverseGroupsGroupId(ctx, id, nil)
			if err != nil {
				return nil, err
			}
			typeIDs = slices.Concat(typeIDs, r2.Types)
		}
		for _, id := range typeIDs {
			_, err := s.GetOrCreateTypeESI(ctx, id)
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

func (s *EveUniverseService) UpdateShipSkills(ctx context.Context) error {
	return s.st.UpdateEveShipSkills(ctx)
}

func (s *EveUniverseService) ListTypeDogmaAttributesForType(
	ctx context.Context,
	typeID int32,
) ([]*app.EveTypeDogmaAttribute, error) {
	return s.st.ListEveTypeDogmaAttributesForType(ctx, typeID)
}
