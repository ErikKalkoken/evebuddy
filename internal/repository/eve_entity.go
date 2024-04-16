package repository

import (
	"context"
	"database/sql"
	"errors"
	"example/evebuddy/internal/api/images"
	"example/evebuddy/internal/helper/set"
	islices "example/evebuddy/internal/helper/slices"
	"example/evebuddy/internal/sqlc"
	"fmt"

	"fyne.io/fyne/v2"
)

type EveEntityCategory int

// Supported categories of EveEntity
const (
	EveEntityUndefined EveEntityCategory = iota
	EveEntityAlliance
	EveEntityCharacter
	EveEntityCorporation
	EveEntityFaction
	EveEntityMailList
)

type EveEntity struct {
	Category EveEntityCategory
	ID       int32
	Name     string
}

func (e *EveEntity) IconURL(size int) (fyne.URI, error) {
	switch e.Category {
	case EveEntityAlliance:
		return images.AllianceLogoURL(e.ID, size)
	case EveEntityCharacter:
		return images.CharacterPortraitURL(e.ID, size)
	case EveEntityCorporation:
		return images.CorporationLogoURL(e.ID, size)
	case EveEntityFaction:
		return images.FactionLogoURL(e.ID, size)
	}
	return nil, errors.New("can not match category")
}

func eveEntityFromDBModel(e sqlc.EveEntity) EveEntity {
	if e.ID == 0 {
		return EveEntity{}
	}
	category := eveEntityCategoryFromDBModel(e.Category)
	return EveEntity{
		Category: category,
		ID:       int32(e.ID),
		Name:     e.Name,
	}
}

// Eve Entity categories in DB models
const (
	eveEntityAllianceCategoryDB    = "alliance"
	eveEntityCharacterCategoryDB   = "character"
	eveEntityCorporationCategoryDB = "corporation"
	eveEntityFactionCategoryDB     = "faction"
	eveEntityMailListCategoryDB    = "mail_list"
)

func eveEntityCategoryFromDBModel(c string) EveEntityCategory {
	categoryMap := map[string]EveEntityCategory{
		eveEntityAllianceCategoryDB:    EveEntityAlliance,
		eveEntityCharacterCategoryDB:   EveEntityCharacter,
		eveEntityCorporationCategoryDB: EveEntityCorporation,
		eveEntityFactionCategoryDB:     EveEntityFaction,
		eveEntityMailListCategoryDB:    EveEntityMailList,
	}
	c2, ok := categoryMap[c]
	if !ok {
		panic(fmt.Sprintf("Can not map unknown category: %s", c))
	}
	return c2
}

func eveEntityDBModelCategoryFromCategory(c EveEntityCategory) string {
	categoryMap := map[EveEntityCategory]string{
		EveEntityAlliance:    eveEntityAllianceCategoryDB,
		EveEntityCharacter:   eveEntityCharacterCategoryDB,
		EveEntityCorporation: eveEntityCorporationCategoryDB,
		EveEntityFaction:     eveEntityFactionCategoryDB,
		EveEntityMailList:    eveEntityMailListCategoryDB,
	}
	c2, ok := categoryMap[c]
	if !ok {
		panic(fmt.Sprintf("Can not map unknown category: %v", c))
	}
	return c2
}

func (r *Repository) CreateEveEntity(ctx context.Context, id int32, name string, category EveEntityCategory) (EveEntity, error) {
	arg := sqlc.CreateEveEntityParams{
		ID:       int64(id),
		Category: eveEntityDBModelCategoryFromCategory(category),
		Name:     name,
	}
	e, err := r.q.CreateEveEntity(ctx, arg)
	if err != nil {
		return EveEntity{}, fmt.Errorf("failed to create eve entity %v, %w", arg, err)
	}
	return eveEntityFromDBModel(e), nil
}

func (r *Repository) GetEveEntity(ctx context.Context, id int32) (EveEntity, error) {
	e, err := r.q.GetEveEntity(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return EveEntity{}, fmt.Errorf("failed to get EveEntity for id %d: %w", id, err)
	}
	e2 := eveEntityFromDBModel(e)
	return e2, nil
}

func (r *Repository) GetEveEntityByNameAndCategory(ctx context.Context, name string, category EveEntityCategory) (EveEntity, error) {
	arg := sqlc.GetEveEntityByNameAndCategoryParams{
		Name:     name,
		Category: eveEntityDBModelCategoryFromCategory(category),
	}
	e, err := r.q.GetEveEntityByNameAndCategory(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return EveEntity{}, err
	}
	e2 := eveEntityFromDBModel(e)
	return e2, nil
}

func (r *Repository) ListEveEntitiesByPartialName(ctx context.Context, partial string) ([]EveEntity, error) {
	ee, err := r.q.ListEveEntitiesByPartialName(ctx, fmt.Sprintf("%%%s%%", partial))
	if err != nil {
		return nil, err
	}
	ee2 := make([]EveEntity, len(ee))
	for i, e := range ee {
		ee2[i] = eveEntityFromDBModel(e)
	}
	return ee2, nil
}

func (r *Repository) ListEveEntityIDs(ctx context.Context) ([]int32, error) {
	ids, err := r.q.ListEveEntityIDs(ctx)
	if err != nil {
		return nil, err
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}

func (r *Repository) ListEveEntitiesByName(ctx context.Context, name string) ([]EveEntity, error) {
	ee, err := r.q.ListEveEntitiesByName(ctx, name)
	if err != nil {
		return nil, err
	}
	ee2 := make([]EveEntity, len(ee))
	for i, e := range ee {
		ee2[i] = eveEntityFromDBModel(e)
	}
	return ee2, nil
}

func (r *Repository) MissingEveEntityIDs(ctx context.Context, ids []int32) (*set.Set[int32], error) {
	currentIDs, err := r.ListEveEntityIDs(ctx)
	if err != nil {
		return nil, err
	}
	current := set.NewFromSlice(currentIDs)
	incoming := set.NewFromSlice(ids)
	missing := incoming.Difference(current)
	return missing, nil
}

func (r *Repository) UpdateOrCreateEveEntity(ctx context.Context, id int32, name string, category EveEntityCategory) (EveEntity, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return EveEntity{}, err
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
			return EveEntity{}, err
		}
		arg := sqlc.UpdateEveEntityParams{
			ID:       int64(id),
			Name:     name,
			Category: categoryDB,
		}
		e, err = qtx.UpdateEveEntity(ctx, arg)
		if err != nil {
			return EveEntity{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return EveEntity{}, err
	}
	return eveEntityFromDBModel(e), nil
}
