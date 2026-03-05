package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateCharacterJumpCloneParams struct {
	CharacterID int64
	JumpCloneID int64
	Implants    []int64
	LocationID  int64
	Name        optional.Optional[string]
}

func (st *Storage) CreateCharacterJumpClone(ctx context.Context, arg CreateCharacterJumpCloneParams) error {
	return createCharacterJumpClone(ctx, st.qRW, arg)
}

func (st *Storage) GetCharacterJumpClone(ctx context.Context, characterID int64, cloneID int64) (*app.CharacterJumpClone, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterJumpClone: %d %d: %w", characterID, cloneID, err)
	}
	if characterID == 0 || cloneID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCharacterJumpClone(ctx, queries.GetCharacterJumpCloneParams{
		CharacterID: characterID,
		JumpCloneID: cloneID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	o := characterJumpCloneFromDBModel(characterJumpCloneFromDBModelParams{
		jc:               r.CharacterJumpClone,
		locationName:     r.LocationName,
		regionID:         r.RegionID,
		regionName:       r.RegionName,
		locationSecurity: r.LocationSecurity,
	})
	x, err := listCharacterJumpCloneImplants(ctx, st.qRO, o.ID)
	if err != nil {
		return nil, err
	}
	o.Implants = x
	return o, nil
}

func listCharacterJumpCloneImplants(ctx context.Context, q *queries.Queries, cloneID int64) ([]*app.CharacterJumpCloneImplant, error) {
	rows, err := q.ListCharacterJumpCloneImplant(ctx, queries.ListCharacterJumpCloneImplantParams{
		CloneID:          cloneID,
		DogmaAttributeID: app.EveDogmaAttributeImplantSlot,
	})
	if err != nil {
		return nil, fmt.Errorf("get character jump clone implants for clone ID %d: %w", cloneID, err)
	}
	var oo []*app.CharacterJumpCloneImplant
	for _, r := range rows {
		oo = append(oo, characterJumpCloneImplantFromDBModel(characterJumpCloneImplantFromDBModelParams{
			jci:  r.CharacterJumpCloneImplant,
			et:   r.EveType,
			eg:   r.EveGroup,
			ec:   r.EveCategory,
			slot: r.SlotNum,
		}))
	}
	return oo, nil
}

type characterJumpCloneImplantFromDBModelParams struct {
	jci  queries.CharacterJumpCloneImplant
	et   queries.EveType
	eg   queries.EveGroup
	ec   queries.EveCategory
	slot sql.NullFloat64
}

func characterJumpCloneImplantFromDBModel(arg characterJumpCloneImplantFromDBModelParams) *app.CharacterJumpCloneImplant {
	if arg.jci.CloneID == 0 {
		panic("missing clone ID")
	}
	o2 := &app.CharacterJumpCloneImplant{
		EveType: eveTypeFromDBModel(arg.et, arg.eg, arg.ec),
		ID:      arg.jci.ID,
	}
	if arg.slot.Valid {
		o2.SlotNum = int(arg.slot.Float64)
	}
	return o2
}

// TODO: Refactor SQL for better performance

func (st *Storage) ListAllCharacterJumpClones(ctx context.Context) ([]*app.CharacterJumpClone2, error) {
	rows, err := st.qRO.ListAllCharacterJumpClones(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all character jump clones: %w", err)
	}
	var oo []*app.CharacterJumpClone2
	for _, r := range rows {
		el, err := st.eveLocationFromDBModel(ctx, queries.EveLocation{
			ID:               r.LocationID,
			EveSolarSystemID: r.LocationSolarSystemID,
			EveTypeID:        r.LocationTypeID,
			Name:             r.LocationName,
			OwnerID:          r.LocationOwnerID,
		})
		if err != nil {
			return nil, err
		}
		oo = append(oo, &app.CharacterJumpClone2{
			ID:            r.ID,
			ImplantsCount: int(r.ImplantsCount),
			CloneID:       r.JumpCloneID,
			Character:     &app.EntityShort{ID: r.CharacterID, Name: r.CharacterName},
			Location:      el,
		})
	}
	return oo, nil
}

func (st *Storage) ListCharacterJumpClones(ctx context.Context, characterID int64) ([]*app.CharacterJumpClone, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterJumpClones: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterJumpClones(ctx, characterID)
	if err != nil {
		return nil, wrapErr(err)
	}
	var oo []*app.CharacterJumpClone
	for _, r := range rows {
		o := characterJumpCloneFromDBModel(characterJumpCloneFromDBModelParams{
			jc:               r.CharacterJumpClone,
			locationName:     r.LocationName,
			regionID:         r.RegionID,
			regionName:       r.RegionName,
			locationSecurity: r.LocationSecurity,
		})
		x, err := listCharacterJumpCloneImplants(ctx, st.qRO, r.CharacterJumpClone.ID)
		if err != nil {
			return nil, err
		}
		o.Implants = x
		oo = append(oo, o)
	}
	return oo, nil
}

func (st *Storage) ReplaceCharacterJumpClones(ctx context.Context, characterID int64, args []CreateCharacterJumpCloneParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("replaceCharacterJumpClones for ID %d: %+v: %w", characterID, args, err)
	}
	if characterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	if err := qtx.DeleteCharacterJumpClones(ctx, characterID); err != nil {
		return wrapErr(err)
	}
	for _, arg := range args {
		if err := createCharacterJumpClone(ctx, qtx, arg); err != nil {
			return wrapErr(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	return nil
}

func createCharacterJumpClone(ctx context.Context, q *queries.Queries, arg CreateCharacterJumpCloneParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("createCharacterJumpClone: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.JumpCloneID == 0 || arg.LocationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	arg2 := queries.CreateCharacterJumpCloneParams{
		CharacterID: arg.CharacterID,
		JumpCloneID: arg.JumpCloneID,
		LocationID:  arg.LocationID,
		Name:        arg.Name.ValueOrZero(),
	}
	cloneID, err := q.CreateCharacterJumpClone(ctx, arg2)
	if err != nil {
		return wrapErr(err)
	}
	for _, eveTypeID := range arg.Implants {
		arg3 := queries.CreateCharacterJumpCloneImplantParams{
			CloneID:   cloneID,
			EveTypeID: eveTypeID,
		}
		if err := q.CreateCharacterJumpCloneImplant(ctx, arg3); err != nil {
			return wrapErr(err)
		}
	}
	return nil
}

type characterJumpCloneFromDBModelParams struct {
	jc               queries.CharacterJumpClone
	locationName     string
	locationSecurity sql.NullFloat64
	regionID         sql.NullInt64
	regionName       sql.NullString
}

func characterJumpCloneFromDBModel(arg characterJumpCloneFromDBModelParams) *app.CharacterJumpClone {
	if arg.jc.CharacterID == 0 || arg.jc.JumpCloneID == 0 || arg.jc.LocationID == 0 {
		panic("invalid IDs")
	}
	o2 := &app.CharacterJumpClone{
		CharacterID: arg.jc.CharacterID,
		ID:          arg.jc.ID,
		CloneID:     arg.jc.JumpCloneID,
		Location: &app.EveLocationShort{
			ID:             arg.jc.LocationID,
			Name:           optional.New(arg.locationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(arg.locationSecurity)},
		Name: optional.FromZeroValue(arg.jc.Name),
	}
	if arg.regionID.Valid && arg.regionName.Valid {
		o2.Region = &app.EntityShort{ID: arg.regionID.Int64, Name: arg.regionName.String}
	}
	return o2
}
