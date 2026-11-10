package storage

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

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

// ReplaceCorporationMembers replaces the members of a corporation with characterIDs.
// It reports whether anything was changed.
func (st *Storage) ReplaceCorporationMembers(ctx context.Context, corporationID int64, characterIDs set.Set[int64]) (bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceCorporationMembers: corporation %d: %w", corporationID, err)
	}
	if corporationID == 0 || characterIDs.Contains(0) {
		return false, wrapErr(app.ErrInvalid)
	}
	current, err := st.ListCorporationMemberIDs(ctx, corporationID)
	if err != nil {
		return false, wrapErr(err)
	}
	added := set.Difference(characterIDs, current)
	removed := set.Difference(current, characterIDs)
	if added.Size() == 0 && removed.Size() == 0 {
		return false, nil
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return false, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	for ids := range slices.Chunk(slices.Collect(removed.All()), st.MaxIDsPerQuery) {
		err := qtx.DeleteCorporationMembers(ctx, queries.DeleteCorporationMembersParams{
			CorporationID: corporationID,
			CharacterIds:  ids,
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("delete %d members: %w", len(ids), err))
		}
	}
	for characterID := range added.All() {
		err := qtx.CreateCorporationMember(ctx, queries.CreateCorporationMemberParams{
			CharacterID:   characterID,
			CorporationID: corporationID,
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("create member %d: %w", characterID, err))
		}
	}
	if err := tx.Commit(); err != nil {
		return false, wrapErr(err)
	}
	slog.Info("Corporation members replaced", "corporationID", corporationID, "added", added.Size(), "removed", removed.Size())
	return true, nil
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
	return set.Collect(slices.Values(ids)), nil
}

func corporationMemberFromDBModel(o queries.CorporationMember, ee queries.EveEntity) *app.CorporationMember {
	o2 := &app.CorporationMember{
		CorporationID: o.CorporationID,
		Character:     eveEntityFromDBModel(ee),
	}
	return o2
}
