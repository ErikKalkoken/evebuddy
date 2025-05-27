package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

// FIXME: Wrong unique clause and missing index for ref_id

type CreateCharacterWalletJournalEntryParams struct {
	Amount        float64
	Balance       float64
	ContextID     int64
	ContextIDType string
	Date          time.Time
	Description   string
	FirstPartyID  int32
	RefID         int64
	CharacterID   int32
	Reason        string
	RefType       string
	SecondPartyID int32
	Tax           float64
	TaxReceiverID int32
}

func (st *Storage) CreateCharacterWalletJournalEntry(ctx context.Context, arg CreateCharacterWalletJournalEntryParams) error {
	if arg.CharacterID == 0 || arg.RefID == 0 {
		return fmt.Errorf("CreateCharacterWalletJournalEntry: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateCharacterWalletJournalEntryParams{
		Amount:        arg.Amount,
		Balance:       arg.Balance,
		ContextID:     arg.ContextID,
		ContextIDType: arg.ContextIDType,
		Date:          arg.Date,
		Description:   arg.Description,
		RefID:         arg.RefID,
		CharacterID:   int64(arg.CharacterID),
		RefType:       arg.RefType,
		Reason:        arg.Reason,
		Tax:           arg.Tax,
	}
	if arg.FirstPartyID != 0 {
		arg2.FirstPartyID.Int64 = int64(arg.FirstPartyID)
		arg2.FirstPartyID.Valid = true
	}
	if arg.SecondPartyID != 0 {
		arg2.SecondPartyID.Int64 = int64(arg.SecondPartyID)
		arg2.SecondPartyID.Valid = true
	}
	if arg.TaxReceiverID != 0 {
		arg2.TaxReceiverID.Int64 = int64(arg.TaxReceiverID)
		arg2.TaxReceiverID.Valid = true
	}
	err := st.qRW.CreateCharacterWalletJournalEntry(ctx, arg2)
	if err != nil {
		return fmt.Errorf("create wallet journal entry for character %d: %w", arg.CharacterID, err)
	}
	return nil
}

func (st *Storage) GetCharacterWalletJournalEntry(ctx context.Context, characterID int32, refID int64) (*app.CharacterWalletJournalEntry, error) {
	arg := queries.GetCharacterWalletJournalEntryParams{
		CharacterID: int64(characterID),
		RefID:       refID,
	}
	r, err := st.qRO.GetCharacterWalletJournalEntry(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get wallet journal entry for character %d: %w", characterID, convertGetError(err))
	}
	o := r.CharacterWalletJournalEntry
	firstParty := nullEveEntry{ID: o.FirstPartyID, Name: r.FirstName, Category: r.FirstCategory}
	secondParty := nullEveEntry{ID: o.SecondPartyID, Name: r.SecondName, Category: r.SecondCategory}
	taxReceiver := nullEveEntry{ID: o.TaxReceiverID, Name: r.TaxName, Category: r.TaxCategory}
	return characterWalletJournalEntryFromDBModel(o, firstParty, secondParty, taxReceiver), err
}

func (st *Storage) ListCharacterWalletJournalEntryIDs(ctx context.Context, characterID int32) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterWalletJournalEntryRefIDs(ctx, int64(characterID))
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list wallet journal entry ids for character %d: %w", characterID, err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListCharacterWalletJournalEntries(ctx context.Context, id int32) ([]*app.CharacterWalletJournalEntry, error) {
	rows, err := st.qRO.ListCharacterWalletJournalEntries(ctx, int64(id))
	if err != nil {
		return nil, fmt.Errorf("list wallet journal entries for character %d: %w", id, err)
	}
	ee := make([]*app.CharacterWalletJournalEntry, len(rows))
	for i, r := range rows {
		o := r.CharacterWalletJournalEntry
		firstParty := nullEveEntry{ID: o.FirstPartyID, Name: r.FirstName, Category: r.FirstCategory}
		secondParty := nullEveEntry{ID: o.SecondPartyID, Name: r.SecondName, Category: r.SecondCategory}
		taxReceiver := nullEveEntry{ID: o.TaxReceiverID, Name: r.TaxName, Category: r.TaxCategory}
		ee[i] = characterWalletJournalEntryFromDBModel(o, firstParty, secondParty, taxReceiver)
	}
	return ee, nil
}

func characterWalletJournalEntryFromDBModel(
	o queries.CharacterWalletJournalEntry,
	firstParty, secondParty, taxReceiver nullEveEntry,
) *app.CharacterWalletJournalEntry {
	o2 := &app.CharacterWalletJournalEntry{
		Amount:        o.Amount,
		Balance:       o.Balance,
		ContextID:     o.ContextID,
		ContextIDType: o.ContextIDType,
		Date:          o.Date,
		Description:   o.Description,
		FirstParty:    eveEntityFromNullableDBModel(firstParty),
		ID:            o.ID,
		RefID:         o.RefID,
		CharacterID:   int32(o.CharacterID),
		Reason:        o.Reason,
		RefType:       o.RefType,
		SecondParty:   eveEntityFromNullableDBModel(secondParty),
		Tax:           o.Tax,
		TaxReceiver:   eveEntityFromNullableDBModel(taxReceiver),
	}
	return o2
}
