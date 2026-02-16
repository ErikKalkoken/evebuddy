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

func (st *Storage) DeleteCharacterLoyaltyPointEntries(ctx context.Context, characterID int64, corporationIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCharacterLoyaltyPointEntriesByID for character %d and job IDs: %v: %w", characterID, corporationIDs, err)
	}
	if characterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if corporationIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.DeleteCharacterLoyaltyPointEntries(ctx, queries.DeleteCharacterLoyaltyPointEntriesParams{
		CharacterID:    characterID,
		CorporationIds: slices.Collect(corporationIDs.All()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Loyalty Points Entries deleted", "characterID", characterID, "corporationIDs", corporationIDs)
	return nil
}

func (st *Storage) GetCharacterLoyaltyPointEntry(ctx context.Context, characterID int64, corporationID int64) (*app.CharacterLoyaltyPointEntry, error) {
	r, err := st.qRO.GetCharacterLoyaltyPointEntry(ctx, queries.GetCharacterLoyaltyPointEntryParams{
		CharacterID:   characterID,
		CorporationID: corporationID,
	})
	if err != nil {
		return nil, fmt.Errorf("GetCharacterLoyaltyPointEntry for character %d: %w", characterID, convertGetError(err))
	}
	o := characterLoyaltyPointEntryFromDBModel(characterLoyaltyPointEntryFromDBModelParams{
		entry:           r.CharacterLoyaltyPointEntry,
		corporationName: r.CorporationName,
		faction: nullEveEntry{
			id:       r.FactionID,
			category: r.FactionCategory,
			name:     r.FactionName,
		},
	})

	return o, err
}

func (st *Storage) ListAllCharacterLoyaltyPointEntries(ctx context.Context) ([]*app.CharacterLoyaltyPointEntry, error) {
	rows, err := st.qRO.ListAllCharacterLoyaltyPointEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListAllCharacterLoyaltyPointEntries: %w", err)
	}
	var oo []*app.CharacterLoyaltyPointEntry
	for _, r := range rows {
		oo = append(oo, characterLoyaltyPointEntryFromDBModel(characterLoyaltyPointEntryFromDBModelParams{
			entry:           r.CharacterLoyaltyPointEntry,
			corporationName: r.CorporationName,
			faction: nullEveEntry{
				id:       r.FactionID,
				category: r.FactionCategory,
				name:     r.FactionName,
			},
		}))
	}
	return oo, nil
}

func (st *Storage) ListCharacterLoyaltyPointEntryIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterLoyaltyPointEntryIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("ListCharacterLoyaltyPointEntryIDs for character %d: %w", characterID, err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListCharacterLoyaltyPointEntries(ctx context.Context, characterID int64) ([]*app.CharacterLoyaltyPointEntry, error) {
	rows, err := st.qRO.ListCharacterLoyaltyPointEntries(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("ListCharacterLoyaltyPointEntry for character %d: %w", characterID, err)
	}
	var oo []*app.CharacterLoyaltyPointEntry
	for _, r := range rows {
		oo = append(oo, characterLoyaltyPointEntryFromDBModel(characterLoyaltyPointEntryFromDBModelParams{
			entry:           r.CharacterLoyaltyPointEntry,
			corporationName: r.CorporationName,
			faction: nullEveEntry{
				id:       r.FactionID,
				category: r.FactionCategory,
				name:     r.FactionName,
			},
		}))
	}
	return oo, nil
}

type characterLoyaltyPointEntryFromDBModelParams struct {
	entry           queries.CharacterLoyaltyPointEntry
	corporationName string
	faction         nullEveEntry
}

func characterLoyaltyPointEntryFromDBModel(arg characterLoyaltyPointEntryFromDBModelParams) *app.CharacterLoyaltyPointEntry {
	o2 := &app.CharacterLoyaltyPointEntry{
		CharacterID: arg.entry.CharacterID,
		Corporation: &app.EntityShort[int64]{
			ID:   arg.entry.CorporationID,
			Name: arg.corporationName,
		},
		Faction:       eveEntityFromNullableDBModel(arg.faction),
		LoyaltyPoints: arg.entry.LoyaltyPoints,
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
