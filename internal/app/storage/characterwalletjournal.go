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
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharacterWalletJournalEntry: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.RefID == 0 {
		return wrapErr(app.ErrInvalid)
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
		return wrapErr(err)
	}
	return nil
}

type GetCharacterWalletJournalEntryParams struct {
	CharacterID int32
	RefID       int64
}

func (st *Storage) GetCharacterWalletJournalEntry(ctx context.Context, arg GetCharacterWalletJournalEntryParams) (*app.CharacterWalletJournalEntry, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterWalletJournalEntry: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.RefID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCharacterWalletJournalEntry(ctx, queries.GetCharacterWalletJournalEntryParams{
		CharacterID: int64(arg.CharacterID),
		RefID:       arg.RefID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	o := r.CharacterWalletJournalEntry
	firstParty := nullEveEntry{id: o.FirstPartyID, name: r.FirstName, category: r.FirstCategory}
	secondParty := nullEveEntry{id: o.SecondPartyID, name: r.SecondName, category: r.SecondCategory}
	taxReceiver := nullEveEntry{id: o.TaxReceiverID, name: r.TaxName, category: r.TaxCategory}
	return characterWalletJournalEntryFromDBModel(o, firstParty, secondParty, taxReceiver), err
}

func (st *Storage) ListCharacterWalletJournalEntryIDs(ctx context.Context, characterID int32) (set.Set[int64], error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("list wallet journal entry ids for character %d: %w", characterID, err)
	}
	if characterID == 0 {
		return set.Set[int64]{}, wrapErr(app.ErrInvalid)
	}
	ids, err := st.qRO.ListCharacterWalletJournalEntryRefIDs(ctx, int64(characterID))
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListCharacterWalletJournalEntries(ctx context.Context, characterID int32) ([]*app.CharacterWalletJournalEntry, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("list wallet journal entries for character %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterWalletJournalEntries(ctx, int64(characterID))
	if err != nil {
		return nil, wrapErr(err)
	}
	ee := make([]*app.CharacterWalletJournalEntry, len(rows))
	for i, r := range rows {
		o := r.CharacterWalletJournalEntry
		firstParty := nullEveEntry{id: o.FirstPartyID, name: r.FirstName, category: r.FirstCategory}
		secondParty := nullEveEntry{id: o.SecondPartyID, name: r.SecondName, category: r.SecondCategory}
		taxReceiver := nullEveEntry{id: o.TaxReceiverID, name: r.TaxName, category: r.TaxCategory}
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
