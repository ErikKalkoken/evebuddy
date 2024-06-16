package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

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
	if arg.RefID == 0 {
		return fmt.Errorf("CharacterWalletJournalEntry ID can not be zero, Character %d", arg.CharacterID)
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
	err := st.q.CreateCharacterWalletJournalEntry(ctx, arg2)
	return err
}

func (st *Storage) GetCharacterWalletJournalEntry(ctx context.Context, characterID int32, refID int64) (*app.CharacterWalletJournalEntry, error) {
	arg := queries.GetCharacterWalletJournalEntryParams{
		CharacterID: int64(characterID),
		RefID:       refID,
	}
	row, err := st.q.GetCharacterWalletJournalEntry(ctx, arg)
	if err != nil {
		return nil, err
	}
	return characterWalletJournalEntryFromDBModel(
		row.CharacterWalletJournalEntry,
		row.CharacterWalletJournalEntryFirstParty,
		row.CharacterWalletJournalEntrySecondParty,
		row.CharacterWalletJournalEntryTaxReceiver,
	), err
}

func (st *Storage) ListCharacterWalletJournalEntryIDs(ctx context.Context, characterID int32) ([]int64, error) {
	return st.q.ListCharacterWalletJournalEntryRefIDs(ctx, int64(characterID))
}

func (st *Storage) ListCharacterWalletJournalEntries(ctx context.Context, characterID int32) ([]*app.CharacterWalletJournalEntry, error) {
	rows, err := st.q.ListCharacterWalletJournalEntries(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	ee := make([]*app.CharacterWalletJournalEntry, len(rows))
	for i, row := range rows {
		ee[i] = characterWalletJournalEntryFromDBModel(
			row.CharacterWalletJournalEntry,
			row.CharacterWalletJournalEntryFirstParty,
			row.CharacterWalletJournalEntrySecondParty,
			row.CharacterWalletJournalEntryTaxReceiver)
	}
	return ee, nil
}

func characterWalletJournalEntryFromDBModel(
	o queries.CharacterWalletJournalEntry,
	firstParty queries.CharacterWalletJournalEntryFirstParty,
	secondParty queries.CharacterWalletJournalEntrySecondParty,
	taxReceiver queries.CharacterWalletJournalEntryTaxReceiver,
) *app.CharacterWalletJournalEntry {
	o2 := &app.CharacterWalletJournalEntry{
		Amount:        o.Amount,
		Balance:       o.Balance,
		ContextID:     o.ContextID,
		ContextIDType: o.ContextIDType,
		Date:          o.Date,
		Description:   o.Description,
		FirstParty:    eveEntityFromNullableDBModel(nullEveEntry(firstParty)),
		ID:            o.ID,
		RefID:         o.RefID,
		CharacterID:   int32(o.CharacterID),
		Reason:        o.Reason,
		RefType:       o.RefType,
		SecondParty:   eveEntityFromNullableDBModel(nullEveEntry(secondParty)),
		Tax:           o.Tax,
		TaxReceiver:   eveEntityFromNullableDBModel(nullEveEntry(taxReceiver)),
	}
	return o2
}
