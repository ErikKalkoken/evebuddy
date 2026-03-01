package storage

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateCharacterImplantParams struct {
	CharacterID int64
	TypeID      int64
}

func (st *Storage) CreateCharacterImplant(ctx context.Context, arg CreateCharacterImplantParams) error {
	return createCharacterImplant(ctx, st.qRW, arg)
}

func (st *Storage) GetCharacterImplant(ctx context.Context, characterID int64, typeID int64) (*app.CharacterImplant, error) {
	r, err := st.qRO.GetCharacterImplant(ctx, queries.GetCharacterImplantParams{
		CharacterID:      characterID,
		DogmaAttributeID: app.EveDogmaAttributeImplantSlot,
		EveTypeID:        typeID,
	})
	if err != nil {
		return nil, fmt.Errorf("get implant %d for character %d: %w", typeID, characterID, convertGetError(err))
	}
	o := characterImplantFromDBModel(characterImplantFromDBModelParams{
		ci: r.CharacterImplant,
		et: r.EveType,
		eg: r.EveGroup,
		ec: r.EveCategory,
		sn: r.SlotNum,
	})
	return o, nil
}

func (st *Storage) ListCharacterImplantIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterImplantIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list implants for character ID %d: %w", characterID, err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) ListCharacterImplants(ctx context.Context, characterID int64) ([]*app.CharacterImplant, error) {
	rows, err := st.qRO.ListCharacterImplants(ctx, queries.ListCharacterImplantsParams{
		DogmaAttributeID: app.EveDogmaAttributeImplantSlot,
		CharacterID:      characterID,
	})
	if err != nil {
		return nil, fmt.Errorf("list implants for character ID %d: %w", characterID, err)
	}
	var oo []*app.CharacterImplant
	for _, r := range rows {
		oo = append(oo, characterImplantFromDBModel(characterImplantFromDBModelParams{
			ci: r.CharacterImplant,
			et: r.EveType,
			eg: r.EveGroup,
			ec: r.EveCategory,
			sn: r.SlotNum,
		}))
	}
	return oo, nil
}

func (st *Storage) ListAllCharacterImplants(ctx context.Context) ([]*app.CharacterImplant, error) {
	rows, err := st.qRO.ListAllCharacterImplants(ctx, app.EveDogmaAttributeImplantSlot)
	if err != nil {
		return nil, fmt.Errorf("list implants for all characters: %w", err)
	}
	var oo []*app.CharacterImplant
	for _, r := range rows {
		oo = append(oo, characterImplantFromDBModel(characterImplantFromDBModelParams{
			ci: r.CharacterImplant,
			et: r.EveType,
			eg: r.EveGroup,
			ec: r.EveCategory,
			sn: r.SlotNum,
		}))
	}
	return oo, nil
}

func (st *Storage) ReplaceCharacterImplants(ctx context.Context, characterID int64, implantIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("replaceCharacterImplants for ID %d: %v: %w", characterID, implantIDs, err)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	if err := qtx.DeleteCharacterImplants(ctx, characterID); err != nil {
		return wrapErr(err)
	}
	for id := range implantIDs.All() {
		err := createCharacterImplant(ctx, qtx, CreateCharacterImplantParams{
			CharacterID: characterID,
			TypeID:      id,
		})
		if err != nil {
			return wrapErr(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	return nil
}

func createCharacterImplant(ctx context.Context, q *queries.Queries, arg CreateCharacterImplantParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("createCharacterImplant: %v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.TypeID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := q.CreateCharacterImplant(ctx, queries.CreateCharacterImplantParams{
		CharacterID: arg.CharacterID,
		EveTypeID:   arg.TypeID,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

type characterImplantFromDBModelParams struct {
	ci queries.CharacterImplant
	et queries.EveType
	eg queries.EveGroup
	ec queries.EveCategory
	sn sql.NullFloat64
}

func characterImplantFromDBModel(arg characterImplantFromDBModelParams) *app.CharacterImplant {
	if arg.ci.CharacterID == 0 {
		panic("missing character ID")
	}
	o2 := &app.CharacterImplant{
		CharacterID: arg.ci.CharacterID,
		EveType:     eveTypeFromDBModel(arg.et, arg.eg, arg.ec),
		ID:          arg.ci.ID,
	}
	if arg.sn.Valid {
		o2.SlotNum = int(arg.sn.Float64)
	}
	return o2
}
