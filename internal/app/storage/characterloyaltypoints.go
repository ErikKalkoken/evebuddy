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

func (st *Storage) GetCharacterLoyaltyPointEntry(ctx context.Context, characterID int64, corporationID int64) (*app.CharacterLoyaltyPointEntry, error) {
	r, err := st.qRO.GetCharacterLoyaltyPointEntry(ctx, queries.GetCharacterLoyaltyPointEntryParams{
		CharacterID:   characterID,
		CorporationID: corporationID,
	})
	if err != nil {
		return nil, fmt.Errorf("GetCharacterLoyaltyPointEntry for character %d: %w", characterID, convertGetError(err))
	}
	o := characterLoyaltyPointEntryFromDBModel(
		r.CharacterLoyaltyPointEntry,
		r.CorporationName,
		nullEveEntity{
			id:       r.FactionID,
			category: r.FactionCategory,
			name:     r.FactionName,
		},
	)

	return o, err
}

func (st *Storage) ListAllCharacterLoyaltyPointEntries(ctx context.Context) ([]*app.CharacterLoyaltyPointEntry, error) {
	rows, err := st.qRO.ListAllCharacterLoyaltyPointEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListAllCharacterLoyaltyPointEntries: %w", err)
	}
	oo := make([]*app.CharacterLoyaltyPointEntry, len(rows))
	for i, r := range rows {
		oo[i] = characterLoyaltyPointEntryFromDBModel(
			r.CharacterLoyaltyPointEntry,
			r.CorporationName,
			nullEveEntity{
				id:       r.FactionID,
				category: r.FactionCategory,
				name:     r.FactionName,
			},
		)
	}
	return oo, nil
}

func (st *Storage) ListCharacterLoyaltyPointEntryIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterLoyaltyPointEntryIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("ListCharacterLoyaltyPointEntryIDs for character %d: %w", characterID, err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) ListCharacterLoyaltyPointEntries(ctx context.Context, characterID int64) ([]*app.CharacterLoyaltyPointEntry, error) {
	rows, err := st.qRO.ListCharacterLoyaltyPointEntries(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("ListCharacterLoyaltyPointEntry for character %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterLoyaltyPointEntry, len(rows))
	for i, r := range rows {
		oo[i] = characterLoyaltyPointEntryFromDBModel(
			r.CharacterLoyaltyPointEntry,
			r.CorporationName,
			nullEveEntity{
				id:       r.FactionID,
				category: r.FactionCategory,
				name:     r.FactionName,
			},
		)
	}
	return oo, nil
}

// ReplaceCharacterLoyaltyPointEntries replaces the loyalty points of a character
// with points, which maps corporation IDs to loyalty points.
// It writes only when something changed and reports whether it did.
func (st *Storage) ReplaceCharacterLoyaltyPointEntries(ctx context.Context, characterID int64, points map[int64]int64) (bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceCharacterLoyaltyPointEntries: character %d: %w", characterID, err)
	}
	if characterID == 0 {
		return false, wrapErr(app.ErrInvalid)
	}
	if _, ok := points[0]; ok {
		return false, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterLoyaltyPointStates(ctx, characterID)
	if err != nil {
		return false, wrapErr(err)
	}
	current := make(map[int64]int64, len(rows))
	for _, r := range rows {
		current[r.CorporationID] = r.LoyaltyPoints
	}
	var obsolete []int64
	for id := range current {
		if _, ok := points[id]; !ok {
			obsolete = append(obsolete, id)
		}
	}
	var changed []int64
	for id, p := range points {
		if p2, ok := current[id]; !ok || p != p2 {
			changed = append(changed, id)
		}
	}
	if len(obsolete) == 0 && len(changed) == 0 {
		return false, nil
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return false, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	for ids := range slices.Chunk(obsolete, st.MaxIDsPerQuery) {
		err := qtx.DeleteCharacterLoyaltyPointEntries(ctx, queries.DeleteCharacterLoyaltyPointEntriesParams{
			CharacterID:    characterID,
			CorporationIds: ids,
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("delete %d entries: %w", len(ids), err))
		}
	}
	for _, id := range changed {
		err := qtx.UpdateOrCreateCharacterLoyaltyPointEntry(ctx, queries.UpdateOrCreateCharacterLoyaltyPointEntryParams{
			CharacterID:   characterID,
			CorporationID: id,
			LoyaltyPoints: points[id],
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("update or create entry for corporation %d: %w", id, err))
		}
	}
	if err := tx.Commit(); err != nil {
		return false, wrapErr(err)
	}
	slog.Info("Loyalty points entries replaced", "characterID", characterID, "changed", len(changed), "deleted", len(obsolete))
	return true, nil
}

func characterLoyaltyPointEntryFromDBModel(
	entry queries.CharacterLoyaltyPointEntry,
	corporationName string,
	faction nullEveEntity,

) *app.CharacterLoyaltyPointEntry {
	o2 := &app.CharacterLoyaltyPointEntry{
		CharacterID: entry.CharacterID,
		Corporation: &app.EntityShort{
			ID:   entry.CorporationID,
			Name: corporationName,
		},
		Faction:       eveEntityFromNullableDBModel(faction),
		LoyaltyPoints: entry.LoyaltyPoints,
		ID:            0,
	}
	return o2
}

type UpdateOrCreateCharacterLoyaltyPointEntryParams struct {
	CharacterID   int64
	CorporationID int64
	LoyaltyPoints int64
}

func (st *Storage) UpdateOrCreateCharacterLoyaltyPointEntry(ctx context.Context, arg UpdateOrCreateCharacterLoyaltyPointEntryParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCharacterLoyaltyPointEntry: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.CorporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateCharacterLoyaltyPointEntry(ctx, queries.UpdateOrCreateCharacterLoyaltyPointEntryParams{
		CharacterID:   arg.CharacterID,
		CorporationID: arg.CorporationID,
		LoyaltyPoints: arg.LoyaltyPoints,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}
