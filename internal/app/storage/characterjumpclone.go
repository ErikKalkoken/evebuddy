package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateCharacterJumpCloneParams struct {
	CharacterID int32
	JumpCloneID int64
	Implants    []int32
	LocationID  int64
	Name        string
}

func (st *Storage) CreateCharacterJumpClone(ctx context.Context, arg CreateCharacterJumpCloneParams) error {
	return createCharacterJumpClone(ctx, st.q, arg)
}

func (st *Storage) GetCharacterJumpClone(ctx context.Context, characterID int32, jumpCloneID int32) (*app.CharacterJumpClone, error) {
	arg := queries.GetCharacterJumpCloneParams{
		CharacterID: int64(characterID),
		JumpCloneID: int64(jumpCloneID),
	}
	row, err := st.q.GetCharacterJumpClone(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("get jump clone for character %d: %w", characterID, err)
	}
	o := characterJumpCloneFromDBModel(row.CharacterJumpClone, row.LocationName, row.RegionID, row.RegionName)
	x, err := listCharacterJumpCloneImplants(ctx, st.q, o.ID)
	if err != nil {
		return nil, err
	}
	o.Implants = x
	return o, nil
}

func listCharacterJumpCloneImplants(ctx context.Context, q *queries.Queries, cloneID int64) ([]*app.CharacterJumpCloneImplant, error) {
	arg2 := queries.ListCharacterJumpCloneImplantParams{
		CloneID:          cloneID,
		DogmaAttributeID: app.EveDogmaAttributeImplantSlot,
	}
	row2, err := q.ListCharacterJumpCloneImplant(ctx, arg2)
	if err != nil {
		return nil, fmt.Errorf("get character jump clone implants for clone ID %d: %w", cloneID, err)
	}
	x := make([]*app.CharacterJumpCloneImplant, len(row2))
	for i, row := range row2 {
		x[i] = characterJumpCloneImplantFromDBModel(
			row.CharacterJumpCloneImplant,
			row.EveType,
			row.EveGroup,
			row.EveCategory, row.SlotNum)
	}
	return x, nil
}

func (st *Storage) ListCharacterJumpClones(ctx context.Context, characterID int32) ([]*app.CharacterJumpClone, error) {
	rows, err := st.q.ListCharacterJumpClones(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("get jump clones for character %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterJumpClone, len(rows))
	for i, row := range rows {
		oo[i] = characterJumpCloneFromDBModel(row.CharacterJumpClone, row.LocationName, row.RegionID, row.RegionName)
		x, err := listCharacterJumpCloneImplants(ctx, st.q, row.CharacterJumpClone.ID)
		if err != nil {
			return nil, err
		}
		oo[i].Implants = x
	}
	return oo, nil
}

func (st *Storage) ReplaceCharacterJumpClones(ctx context.Context, characterID int32, args []CreateCharacterJumpCloneParams) error {
	err := func() error {
		tx, err := st.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		qtx := st.q.WithTx(tx)
		if err := qtx.DeleteCharacterJumpClones(ctx, int64(characterID)); err != nil {
			return err
		}
		for _, arg := range args {
			if err := createCharacterJumpClone(ctx, qtx, arg); err != nil {
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		return fmt.Errorf("replace jump clones for character %d: %w", characterID, err)
	}
	return nil
}

func createCharacterJumpClone(ctx context.Context, q *queries.Queries, arg CreateCharacterJumpCloneParams) error {
	if arg.CharacterID == 0 || arg.JumpCloneID == 0 {
		return fmt.Errorf("IDs must not be zero %v", arg)
	}
	arg2 := queries.CreateCharacterJumpCloneParams{
		CharacterID: int64(arg.CharacterID),
		JumpCloneID: int64(arg.JumpCloneID),
		LocationID:  arg.LocationID,
		Name:        arg.Name,
	}
	cloneID, err := q.CreateCharacterJumpClone(ctx, arg2)
	if err != nil {
		return fmt.Errorf("create character jump clone %v, %w", arg, err)
	}
	for _, eveTypeID := range arg.Implants {
		arg3 := queries.CreateCharacterJumpCloneImplantParams{
			CloneID:   cloneID,
			EveTypeID: int64(eveTypeID),
		}
		if err := q.CreateCharacterJumpCloneImplant(ctx, arg3); err != nil {
			return fmt.Errorf("create character jump clone implant %v, %w", arg, err)
		}
	}
	return nil
}

func characterJumpCloneFromDBModel(o queries.CharacterJumpClone, locationName string, regionID sql.NullInt64, regionName sql.NullString) *app.CharacterJumpClone {
	if o.CharacterID == 0 {
		panic("missing character ID")
	}
	o2 := &app.CharacterJumpClone{
		CharacterID: int32(o.CharacterID),
		ID:          o.ID,
		JumpCloneID: int32(o.JumpCloneID),
		Location:    &app.EntityShort[int64]{ID: o.LocationID, Name: locationName},
		Name:        o.Name,
	}
	if regionID.Valid && regionName.Valid {
		o2.Region = &app.EntityShort[int32]{ID: int32(regionID.Int64), Name: regionName.String}
	}
	return o2
}

func characterJumpCloneImplantFromDBModel(
	o queries.CharacterJumpCloneImplant,
	t queries.EveType,
	g queries.EveGroup,
	c queries.EveCategory,
	s sql.NullFloat64,
) *app.CharacterJumpCloneImplant {
	if o.CloneID == 0 {
		panic("missing clone ID")
	}
	o2 := &app.CharacterJumpCloneImplant{
		EveType: eveTypeFromDBModel(t, g, c),
		ID:      o.ID,
	}
	if s.Valid {
		o2.SlotNum = int(s.Float64)
	}
	return o2
}
