package service

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (s *Service) GetEveType(id int32) (*model.EveType, error) {
	ctx := context.Background()
	return s.r.GetEveType(ctx, id)
}

func (s *Service) GetOrCreateEveCategoryESI(id int32) (*model.EveCategory, error) {
	ctx := context.Background()
	return s.getOrCreateEveCategoryESI(ctx, id)
}

func (s *Service) UpdateEveCategoryWithChildrenESI(categoryID int32) error {
	ctx := context.Background()
	key := fmt.Sprintf("UpdateEveCategoryWithChildrenESI-%d", categoryID)
	_, err, _ := s.singleGroup.Do(key, func() (any, error) {
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
			_, err := s.getOrCreateEveTypeESI(ctx, id)
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

func (s *Service) getOrCreateEveCategoryESI(ctx context.Context, id int32) (*model.EveCategory, error) {
	x, err := s.r.GetEveCategory(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return s.createEveCategoryFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (s *Service) createEveCategoryFromESI(ctx context.Context, id int32) (*model.EveCategory, error) {
	key := fmt.Sprintf("createEveCategoryFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		r, _, err := s.esiClient.ESI.UniverseApi.GetUniverseCategoriesCategoryId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveCategoryParams{
			ID:          id,
			Name:        r.Name,
			IsPublished: r.Published,
		}
		return s.r.CreateEveCategory(ctx, arg)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveCategory), nil
}

func (s *Service) GetOrCreateEveGroupESI(id int32) (*model.EveGroup, error) {
	ctx := context.Background()
	return s.getOrCreateEveGroupESI(ctx, id)
}

func (s *Service) getOrCreateEveGroupESI(ctx context.Context, id int32) (*model.EveGroup, error) {
	x, err := s.r.GetEveGroup(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return s.createEveGroupFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (s *Service) createEveGroupFromESI(ctx context.Context, id int32) (*model.EveGroup, error) {
	key := fmt.Sprintf("createEveGroupFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		r, _, err := s.esiClient.ESI.UniverseApi.GetUniverseGroupsGroupId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		c, err := s.getOrCreateEveCategoryESI(ctx, r.CategoryId)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveGroupParams{
			ID:          id,
			Name:        r.Name,
			CategoryID:  c.ID,
			IsPublished: r.Published,
		}
		if err := s.r.CreateEveGroup(ctx, arg); err != nil {
			return nil, err
		}
		return s.r.GetEveGroup(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveGroup), nil
}

func (s *Service) GetOrCreateEveTypeESI(id int32) (*model.EveType, error) {
	ctx := context.Background()
	return s.getOrCreateEveTypeESI(ctx, id)
}

func (s *Service) getOrCreateEveTypeESI(ctx context.Context, id int32) (*model.EveType, error) {
	x, err := s.r.GetEveType(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return s.createEveTypeFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (s *Service) createEveTypeFromESI(ctx context.Context, id int32) (*model.EveType, error) {
	key := fmt.Sprintf("createEveTypeFromESI-%d", id)
	x, err, _ := s.singleGroup.Do(key, func() (any, error) {
		t, _, err := s.esiClient.ESI.UniverseApi.GetUniverseTypesTypeId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		g, err := s.getOrCreateEveGroupESI(ctx, t.GroupId)
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
		if err := s.r.CreateEveType(ctx, arg); err != nil {
			return nil, err
		}
		for _, o := range t.DogmaAttributes {
			arg := storage.CreateEveTypeDogmaAttributeParams{
				DogmaAttributeID: o.AttributeId,
				EveTypeID:        id,
				Value:            o.Value,
			}
			if err := s.r.CreateEveTypeDogmaAttribute(ctx, arg); err != nil {
				return nil, err
			}
		}
		for _, o := range t.DogmaEffects {
			arg := storage.CreateEveTypeDogmaEffectParams{
				DogmaEffectID: o.EffectId,
				EveTypeID:     id,
				IsDefault:     o.IsDefault,
			}
			if err := s.r.CreateEveTypeDogmaEffect(ctx, arg); err != nil {
				return nil, err
			}
		}
		return s.r.GetEveType(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return x.(*model.EveType), nil
}
