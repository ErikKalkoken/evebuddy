package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"example/evebuddy/internal/helper/set"
	islices "example/evebuddy/internal/helper/slices"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/sqlc"
)

// Eve Entity categories in DB models
const (
	eveEntityAllianceCategoryDB    = "alliance"
	eveEntityCharacterCategoryDB   = "character"
	eveEntityCorporationCategoryDB = "corporation"
	eveEntityFactionCategoryDB     = "faction"
	eveEntityMailListCategoryDB    = "mail_list"
)

func eveEntityCategoryFromDBModel(c string) model.EveEntityCategory {
	categoryMap := map[string]model.EveEntityCategory{
		eveEntityAllianceCategoryDB:    model.EveEntityAlliance,
		eveEntityCharacterCategoryDB:   model.EveEntityCharacter,
		eveEntityCorporationCategoryDB: model.EveEntityCorporation,
		eveEntityFactionCategoryDB:     model.EveEntityFaction,
		eveEntityMailListCategoryDB:    model.EveEntityMailList,
	}
	c2, ok := categoryMap[c]
	if !ok {
		panic(fmt.Sprintf("Can not map unknown category: %s", c))
	}
	return c2
}

func eveEntityDBModelCategoryFromCategory(c model.EveEntityCategory) string {
	categoryMap := map[model.EveEntityCategory]string{
		model.EveEntityAlliance:    eveEntityAllianceCategoryDB,
		model.EveEntityCharacter:   eveEntityCharacterCategoryDB,
		model.EveEntityCorporation: eveEntityCorporationCategoryDB,
		model.EveEntityFaction:     eveEntityFactionCategoryDB,
		model.EveEntityMailList:    eveEntityMailListCategoryDB,
	}
	c2, ok := categoryMap[c]
	if !ok {
		panic(fmt.Sprintf("Can not map unknown category: %v", c))
	}
	return c2
}

func eveEntityFromDBModel(e sqlc.EveEntity) model.EveEntity {
	if e.ID == 0 {
		return model.EveEntity{}
	}
	category := eveEntityCategoryFromDBModel(e.Category)
	return model.EveEntity{
		Category: category,
		ID:       int32(e.ID),
		Name:     e.Name,
	}
}

func (r *Storage) CreateEveEntity(ctx context.Context, id int32, name string, category model.EveEntityCategory) (model.EveEntity, error) {
	e, err := func() (model.EveEntity, error) {
		if id == 0 {
			return model.EveEntity{}, fmt.Errorf("invalid ID %d", id)
		}
		arg := sqlc.CreateEveEntityParams{
			ID:       int64(id),
			Category: eveEntityDBModelCategoryFromCategory(category),
			Name:     name,
		}
		e, err := r.q.CreateEveEntity(ctx, arg)
		if err != nil {
			return model.EveEntity{}, fmt.Errorf("failed to create eve entity %v, %w", arg, err)
		}
		return eveEntityFromDBModel(e), nil
	}()
	if err != nil {
		return model.EveEntity{}, fmt.Errorf("failed to create EveEntity %d: %w", id, err)
	}
	return e, nil
}

func (r *Storage) GetEveEntity(ctx context.Context, id int32) (model.EveEntity, error) {
	e, err := r.q.GetEveEntity(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return model.EveEntity{}, fmt.Errorf("failed to get EveEntity for id %d: %w", id, err)
	}
	e2 := eveEntityFromDBModel(e)
	return e2, nil
}

func (r *Storage) ListEveEntityByNameAndCategory(ctx context.Context, name string, category model.EveEntityCategory) ([]model.EveEntity, error) {
	var ee2 []model.EveEntity
	arg := sqlc.ListEveEntityByNameAndCategoryParams{
		Name:     name,
		Category: eveEntityDBModelCategoryFromCategory(category),
	}
	ee, err := r.q.ListEveEntityByNameAndCategory(ctx, arg)
	if err != nil {
		return ee2, fmt.Errorf("failed to get EveEntity by name %s and category %s: %w", name, category, err)
	}
	for _, e := range ee {
		ee2 = append(ee2, eveEntityFromDBModel(e))
	}
	return ee2, nil
}

func (r *Storage) GetOrCreateEveEntity(ctx context.Context, id int32, name string, category model.EveEntityCategory) (model.EveEntity, error) {
	label, err := func() (model.EveEntity, error) {
		var e sqlc.EveEntity
		if id == 0 {
			return model.EveEntity{}, fmt.Errorf("invalid ID %d", id)
		}
		tx, err := r.db.Begin()
		if err != nil {
			return model.EveEntity{}, err
		}
		defer tx.Rollback()
		qtx := r.q.WithTx(tx)
		e, err = qtx.GetEveEntity(ctx, int64(id))
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return model.EveEntity{}, err
			}
			arg := sqlc.CreateEveEntityParams{
				ID:       int64(id),
				Name:     name,
				Category: eveEntityDBModelCategoryFromCategory(category),
			}
			e, err = qtx.CreateEveEntity(ctx, arg)
			if err != nil {
				return model.EveEntity{}, err
			}
		}
		if err := tx.Commit(); err != nil {
			return model.EveEntity{}, err
		}
		return eveEntityFromDBModel(e), nil
	}()
	if err != nil {
		return label, fmt.Errorf("failed to get or create eve entity %d: %w", id, err)
	}
	return label, nil
}

func (r *Storage) ListEveEntitiesByPartialName(ctx context.Context, partial string) ([]model.EveEntity, error) {
	ee, err := r.q.ListEveEntitiesByPartialName(ctx, fmt.Sprintf("%%%s%%", partial))
	if err != nil {
		return nil, fmt.Errorf("failed to list EveEntity by partial name %s: %w", partial, err)
	}
	ee2 := make([]model.EveEntity, len(ee))
	for i, e := range ee {
		ee2[i] = eveEntityFromDBModel(e)
	}
	return ee2, nil
}

func (r *Storage) ListEveEntityIDs(ctx context.Context) ([]int32, error) {
	ids, err := r.q.ListEveEntityIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list EveEntity IDs: %w", err)
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}

func (r *Storage) ListEveEntitiesByName(ctx context.Context, name string) ([]model.EveEntity, error) {
	ee, err := r.q.ListEveEntitiesByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to list EveEntities by name %s: %w", name, err)
	}
	ee2 := make([]model.EveEntity, len(ee))
	for i, e := range ee {
		ee2[i] = eveEntityFromDBModel(e)
	}
	return ee2, nil
}

func (r *Storage) MissingEveEntityIDs(ctx context.Context, ids []int32) (*set.Set[int32], error) {
	currentIDs, err := r.ListEveEntityIDs(ctx)
	if err != nil {
		return nil, err
	}
	current := set.NewFromSlice(currentIDs)
	incoming := set.NewFromSlice(ids)
	missing := incoming.Difference(current)
	return missing, nil
}

func (r *Storage) UpdateOrCreateEveEntity(ctx context.Context, id int32, name string, category model.EveEntityCategory) (model.EveEntity, error) {
	e, err := func() (model.EveEntity, error) {
		if id == 0 {
			return model.EveEntity{}, fmt.Errorf("invalid ID %d", id)
		}
		tx, err := r.db.Begin()
		if err != nil {
			return model.EveEntity{}, err
		}
		defer tx.Rollback()
		qtx := r.q.WithTx(tx)
		categoryDB := eveEntityDBModelCategoryFromCategory(category)
		arg := sqlc.CreateEveEntityParams{
			ID:       int64(id),
			Name:     name,
			Category: categoryDB,
		}
		var e sqlc.EveEntity
		e, err = qtx.CreateEveEntity(ctx, arg)
		if err != nil {
			if !isSqlite3ErrConstraint(err) {
				return model.EveEntity{}, err
			}
			arg := sqlc.UpdateEveEntityParams{
				ID:       int64(id),
				Name:     name,
				Category: categoryDB,
			}
			e, err = qtx.UpdateEveEntity(ctx, arg)
			if err != nil {
				return model.EveEntity{}, err
			}
		}
		if err := tx.Commit(); err != nil {
			return model.EveEntity{}, err
		}
		return eveEntityFromDBModel(e), nil
	}()
	if err != nil {
		return model.EveEntity{}, fmt.Errorf("failed to update or create EveEntity %d: %w", id, err)
	}
	return e, nil
}
