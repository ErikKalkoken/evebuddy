package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (s *Service) GetOrCreateEveCategoryESI(id int32) (*model.EveCategory, error) {
	ctx := context.Background()
	return s.getOrCreateEveCategoryESI(ctx, id)
}

func (s *Service) getOrCreateEveCategoryESI(ctx context.Context, id int32) (*model.EveCategory, error) {
	x, err := s.r.GetEveCategory(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.createEveCategoryFromESI(ctx, id)
		}
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
		return s.r.CreateEveCategory(ctx, id, r.Name, r.Published)
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
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.createEveGroupFromESI(ctx, id)
		}
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
		if err := s.r.CreateEveGroup(ctx, id, c.ID, r.Name, r.Published); err != nil {
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
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.createEveTypeFromESI(ctx, id)
		}
		return x, err
	}
	return x, nil
}

func (s *Service) createEveTypeFromESI(ctx context.Context, id int32) (*model.EveType, error) {
	key := fmt.Sprintf("createEveTypeFromESI-%d", id)
	y, err, _ := s.singleGroup.Do(key, func() (any, error) {
		r, _, err := s.esiClient.ESI.UniverseApi.GetUniverseTypesTypeId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		g, err := s.getOrCreateEveGroupESI(ctx, r.GroupId)
		if err != nil {
			return nil, err
		}
		if err := s.r.CreateEveType(ctx, id, r.Description, g.ID, r.Name, r.Published); err != nil {
			return nil, err
		}
		return s.r.GetEveType(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return y.(*model.EveType), nil
}
