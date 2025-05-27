package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

// Eve Entity categories in DB models
const (
	eveEntityAlliance      = "alliance"
	eveEntityCharacter     = "character"
	eveEntityCorporation   = "corporation"
	eveEntityConstellation = "constellation"
	eveEntityFaction       = "faction"
	eveEntityInventoryType = "inventory_type"
	eveEntityMailList      = "mail_list"
	eveEntityRegion        = "region"
	eveEntitySolarSystem   = "solar_system"
	eveEntityStation       = "station"
	eveEntityUnknown       = "unknown"
)

type CreateEveEntityParams struct {
	ID       int32
	Name     string
	Category app.EveEntityCategory
}

func (arg CreateEveEntityParams) isValid() bool {
	return arg.ID != 0
}

func (st *Storage) CreateEveEntity(ctx context.Context, arg CreateEveEntityParams) (*app.EveEntity, error) {
	if !arg.isValid() {
		return nil, fmt.Errorf("CreateEveEntity: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateEveEntityParams{
		ID:       int64(arg.ID),
		Category: eveEntityDBModelCategoryFromCategory(arg.Category),
		Name:     arg.Name,
	}
	e, err := st.qRW.CreateEveEntity(ctx, arg2)
	if err != nil {
		return nil, fmt.Errorf("CreateEveEntity: %+v, %w", arg2, err)
	}
	return eveEntityFromDBModel(e), nil

}

func (st *Storage) GetOrCreateEveEntity(ctx context.Context, arg CreateEveEntityParams) (*app.EveEntity, error) {
	ee, err := func() (*app.EveEntity, error) {
		var e queries.EveEntity
		if !arg.isValid() {
			return nil, app.ErrInvalid
		}
		tx, err := st.dbRW.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		qtx := st.qRW.WithTx(tx)
		e, err = qtx.GetEveEntity(ctx, int64(arg.ID))
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
			arg2 := queries.CreateEveEntityParams{
				ID:       int64(arg.ID),
				Name:     arg.Name,
				Category: eveEntityDBModelCategoryFromCategory(arg.Category),
			}
			e, err = qtx.CreateEveEntity(ctx, arg2)
			if err != nil {
				return nil, err
			}
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return eveEntityFromDBModel(e), nil
	}()
	if err != nil {
		return nil, fmt.Errorf("GetOrCreateEveEntity: %+v: %w", arg, err)
	}
	return ee, nil
}

func (st *Storage) GetEveEntity(ctx context.Context, id int32) (*app.EveEntity, error) {
	e, err := st.qRO.GetEveEntity(ctx, int64(id))
	if err != nil {
		return nil, fmt.Errorf("get eve entity for id %d: %w", id, convertGetError(err))
	}
	return eveEntityFromDBModel(e), nil
}

func (st *Storage) ListEveEntityByNameAndCategory(ctx context.Context, name string, category app.EveEntityCategory) ([]*app.EveEntity, error) {
	var ee2 []*app.EveEntity
	arg := queries.ListEveEntityByNameAndCategoryParams{
		Name:     name,
		Category: eveEntityDBModelCategoryFromCategory(category),
	}
	ee, err := st.qRO.ListEveEntityByNameAndCategory(ctx, arg)
	if err != nil {
		return ee2, fmt.Errorf("get eve entity by name %s and category %s: %w", name, category, err)
	}
	for _, e := range ee {
		ee2 = append(ee2, eveEntityFromDBModel(e))
	}
	return ee2, nil
}

func (st *Storage) ListEveEntitiesByPartialName(ctx context.Context, partial string) ([]*app.EveEntity, error) {
	ee, err := st.qRO.ListEveEntitiesByPartialName(ctx, fmt.Sprintf("%%%s%%", partial))
	if err != nil {
		return nil, fmt.Errorf("list eve entity by partial name %s: %w", partial, err)
	}
	ee2 := make([]*app.EveEntity, len(ee))
	for i, e := range ee {
		ee2[i] = eveEntityFromDBModel(e)
	}
	return ee2, nil
}

func (st *Storage) ListEveEntityIDs(ctx context.Context) (set.Set[int32], error) {
	ids, err := st.qRO.ListEveEntityIDs(ctx)
	if err != nil {
		return set.Set[int32]{}, fmt.Errorf("list eve entity id: %w", err)
	}
	ids2 := set.Of(convertNumericSlice[int32](ids)...)
	return ids2, nil
}

func (st *Storage) ListEveEntitiesByName(ctx context.Context, name string) ([]*app.EveEntity, error) {
	ee, err := st.qRO.ListEveEntitiesByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("list eve entities by name %s: %w", name, err)
	}
	ee2 := make([]*app.EveEntity, len(ee))
	for i, e := range ee {
		ee2[i] = eveEntityFromDBModel(e)
	}
	return ee2, nil
}

// ListEveEntitiesForIDs returns a slice of EveEntites in the same order as ids.
//
// Returns an error if at least one object can not be found.
func (st *Storage) ListEveEntitiesForIDs(ctx context.Context, ids []int32) ([]*app.EveEntity, error) {
	ee := make([]queries.EveEntity, 0)
	for idsChunk := range slices.Chunk(convertNumericSlice[int64](ids), st.MaxListEveEntitiesForIDs) {
		r, err := st.qRO.ListEveEntitiesForIDs(ctx, idsChunk)
		if err != nil {
			return nil, fmt.Errorf("list eve entities for %d ids: %w", len(idsChunk), err)
		}
		ee = slices.Concat(ee, r)
	}
	m := make(map[int32]*app.EveEntity)
	for _, e := range ee {
		m[int32(e.ID)] = eveEntityFromDBModel(e)
	}
	oo := make([]*app.EveEntity, 0)
	for _, id := range ids {
		o, found := m[id]
		if !found {
			return nil, app.ErrNotFound
		}
		oo = append(oo, o)
	}
	return oo, nil
}

// MissingEveEntityIDs returns the IDs, which are have no respective EveEntity in the database.
// IDs with value 0 are ignored.
func (st *Storage) MissingEveEntityIDs(ctx context.Context, ids set.Set[int32]) (set.Set[int32], error) {
	incoming := ids.Clone()
	incoming.Delete(0)
	current, err := st.ListEveEntityIDs(ctx)
	if err != nil {
		return set.Of[int32](), err
	}
	missing := set.Difference(incoming, current)
	return missing, nil
}

func (st *Storage) UpdateOrCreateEveEntity(ctx context.Context, id int32, name string, category app.EveEntityCategory) (*app.EveEntity, error) {
	if id == 0 {
		return nil, fmt.Errorf("can't update or create eve entity with ID %d", id)
	}
	categoryDB := eveEntityDBModelCategoryFromCategory(category)
	arg := queries.UpdateOrCreateEveEntityParams{
		ID:       int64(id),
		Name:     name,
		Category: categoryDB,
	}
	e, err := st.qRW.UpdateOrCreateEveEntity(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("update or create eve entity %d: %w", id, err)
	}
	slog.Info("Stored updated Eve Entities", "ID", id)
	return eveEntityFromDBModel(e), nil
}

func eveEntityCategoryFromDBModel(c string) app.EveEntityCategory {
	categoryMap := map[string]app.EveEntityCategory{
		eveEntityAlliance:      app.EveEntityAlliance,
		eveEntityCharacter:     app.EveEntityCharacter,
		eveEntityConstellation: app.EveEntityConstellation,
		eveEntityCorporation:   app.EveEntityCorporation,
		eveEntityFaction:       app.EveEntityFaction,
		eveEntityMailList:      app.EveEntityMailList,
		eveEntityInventoryType: app.EveEntityInventoryType,
		eveEntityRegion:        app.EveEntityRegion,
		eveEntitySolarSystem:   app.EveEntitySolarSystem,
		eveEntityStation:       app.EveEntityStation,
		eveEntityUnknown:       app.EveEntityUnknown,
	}
	c2, ok := categoryMap[c]
	if !ok {
		return app.EveEntityUnknown
	}
	return c2
}

func eveEntityDBModelCategoryFromCategory(c app.EveEntityCategory) string {
	categoryMap := map[app.EveEntityCategory]string{
		app.EveEntityAlliance:      eveEntityAlliance,
		app.EveEntityCharacter:     eveEntityCharacter,
		app.EveEntityConstellation: eveEntityConstellation,
		app.EveEntityCorporation:   eveEntityCorporation,
		app.EveEntityFaction:       eveEntityFaction,
		app.EveEntityMailList:      eveEntityMailList,
		app.EveEntityInventoryType: eveEntityInventoryType,
		app.EveEntityRegion:        eveEntityRegion,
		app.EveEntitySolarSystem:   eveEntitySolarSystem,
		app.EveEntityStation:       eveEntityStation,
		app.EveEntityUnknown:       eveEntityUnknown,
	}
	c2, ok := categoryMap[c]
	if !ok {
		return eveEntityUnknown
	}
	return c2
}

func eveEntityFromDBModel(e queries.EveEntity) *app.EveEntity {
	if e.ID == 0 {
		return nil
	}
	category := eveEntityCategoryFromDBModel(e.Category)
	return &app.EveEntity{
		Category: category,
		ID:       int32(e.ID),
		Name:     e.Name,
	}
}

type nullEveEntry struct {
	ID       sql.NullInt64
	Category sql.NullString
	Name     sql.NullString
}

func eveEntityFromNullableDBModel(e nullEveEntry) *app.EveEntity {
	if !e.ID.Valid {
		return nil
	}
	category := eveEntityCategoryFromDBModel(e.Category.String)
	return &app.EveEntity{
		Category: category,
		ID:       int32(e.ID.Int64),
		Name:     e.Name.String,
	}
}
