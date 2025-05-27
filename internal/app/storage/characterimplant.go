package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateCharacterImplantParams struct {
	CharacterID int32
	EveTypeID   int32
}

func (st *Storage) CreateCharacterImplant(ctx context.Context, arg CreateCharacterImplantParams) error {
	return createCharacterImplant(ctx, st.qRW, arg)
}

func (st *Storage) GetCharacterImplant(ctx context.Context, characterID int32, typeID int32) (*app.CharacterImplant, error) {
	arg := queries.GetCharacterImplantParams{
		CharacterID:      int64(characterID),
		DogmaAttributeID: app.EveDogmaAttributeImplantSlot,
		EveTypeID:        int64(typeID),
	}
	row, err := st.qRO.GetCharacterImplant(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get implant %d for character %d: %w", typeID, characterID, convertGetError(err))
	}
	t2 := characterImplantFromDBModel(
		row.CharacterImplant,
		row.EveType,
		row.EveGroup,
		row.EveCategory, row.SlotNum)
	return t2, nil
}

func (st *Storage) ListCharacterImplants(ctx context.Context, characterID int32) ([]*app.CharacterImplant, error) {
	arg := queries.ListCharacterImplantsParams{
		DogmaAttributeID: app.EveDogmaAttributeImplantSlot,
		CharacterID:      int64(characterID),
	}
	rows, err := st.qRO.ListCharacterImplants(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list implants for character ID %d: %w", characterID, err)
	}
	ii2 := make([]*app.CharacterImplant, len(rows))
	for i, row := range rows {
		ii2[i] = characterImplantFromDBModel(
			row.CharacterImplant,
			row.EveType,
			row.EveGroup,
			row.EveCategory,
			row.SlotNum)
	}
	return ii2, nil
}

func (st *Storage) ReplaceCharacterImplants(ctx context.Context, characterID int32, args []CreateCharacterImplantParams) error {
	err := func() error {
		tx, err := st.dbRW.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		qtx := st.qRW.WithTx(tx)
		if err := qtx.DeleteCharacterImplants(ctx, int64(characterID)); err != nil {
			return err
		}
		for _, arg := range args {
			err := createCharacterImplant(ctx, qtx, arg)
			if err != nil {
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		return fmt.Errorf("replace implants for character ID %d: %w", characterID, err)
	}
	return nil
}

func createCharacterImplant(ctx context.Context, q *queries.Queries, arg CreateCharacterImplantParams) error {
	if arg.CharacterID == 0 || arg.EveTypeID == 0 {
		return fmt.Errorf("createCharacterImplant: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateCharacterImplantParams{
		CharacterID: int64(arg.CharacterID),
		EveTypeID:   int64(arg.EveTypeID),
	}
	err := q.CreateCharacterImplant(ctx, arg2)
	if err != nil {
		return fmt.Errorf("create character implant %v, %w", arg, err)
	}
	return nil
}

func characterImplantFromDBModel(
	o queries.CharacterImplant,
	t queries.EveType,
	g queries.EveGroup,
	c queries.EveCategory,
	s sql.NullFloat64,
) *app.CharacterImplant {
	if o.CharacterID == 0 {
		panic("missing character ID")
	}
	o2 := &app.CharacterImplant{
		CharacterID: int32(o.CharacterID),
		EveType:     eveTypeFromDBModel(t, g, c),
		ID:          o.ID,
	}
	if s.Valid {
		o2.SlotNum = int(s.Float64)
	}
	return o2
}
