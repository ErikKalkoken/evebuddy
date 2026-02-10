package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CorporationMemberParams struct {
	CharacterID   int64
	CorporationID int64
}

func (x CorporationMemberParams) isValid() bool {
	return x.CharacterID != 0 && x.CorporationID != 0
}

func (st *Storage) CreateCorporationMember(ctx context.Context, arg CorporationMemberParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCorporationMember %+v: %w", arg, err)
	}
	if !arg.isValid() {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.CreateCorporationMember(ctx, queries.CreateCorporationMemberParams{
		CharacterID:   arg.CharacterID,
		CorporationID: arg.CorporationID,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetCorporationMember(ctx context.Context, arg CorporationMemberParams) (*app.CorporationMember, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCorporationMember %+v: %w", arg, err)
	}
	if !arg.isValid() {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCorporationMembers(ctx, queries.GetCorporationMembersParams{
		CorporationID: arg.CorporationID,
		CharacterID:   arg.CharacterID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	return corporationMemberFromDBModel(r.CorporationMember, r.EveEntity), nil
}

func (st *Storage) DeleteCorporationMembers(ctx context.Context, corporationID int64, characterIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCorporationMembers %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if characterIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.DeleteCorporationMembers(ctx, queries.DeleteCorporationMembersParams{
		CorporationID: corporationID,
		CharacterIds:  convertNumericSet[int64](characterIDs),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) ListCorporationMembers(ctx context.Context, corporationID int64) ([]*app.CorporationMember, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationMembers for id %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCorporationMembers(ctx, corporationID)
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]*app.CorporationMember, len(rows))
	for i, r := range rows {
		oo[i] = corporationMemberFromDBModel(r.CorporationMember, r.EveEntity)
	}
	return oo, nil
}

func (st *Storage) ListCorporationMemberIDs(ctx context.Context, corporationID int64) (set.Set[int64], error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationMemberIDs for id %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return set.Set[int64]{}, wrapErr(app.ErrInvalid)
	}
	ids, err := st.qRO.ListCorporationMemberIDs(ctx, corporationID)
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	return set.Of(ids...), nil
}

func corporationMemberFromDBModel(o queries.CorporationMember, ee queries.EveEntity) *app.CorporationMember {
	o2 := &app.CorporationMember{
		CorporationID: o.CorporationID,
		Character:     eveEntityFromDBModel(ee),
	}
	return o2
}
