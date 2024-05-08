package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateWalletJournalEntryParams struct {
	Amount        float64
	Balance       float64
	ContextID     int64
	ContextIDType string
	Date          time.Time
	Description   string
	FirstPartyID  int32
	ID            int64
	MyCharacterID int32
	Reason        string
	RefType       string
	SecondPartyID int32
	Tax           float64
	TaxReceiverID int32
}

func (r *Storage) CreateWalletJournalEntry(ctx context.Context, arg CreateWalletJournalEntryParams) error {
	if arg.ID == 0 {
		return fmt.Errorf("WalletJournalEntry ID can not be zero, Character %d", arg.MyCharacterID)
	}
	arg2 := queries.CreateWalletJournalEntriesParams{
		Date:          arg.Date,
		Description:   arg.Description,
		ID:            arg.ID,
		MyCharacterID: int64(arg.MyCharacterID),
		RefType:       arg.RefType,
	}
	if arg.Amount != 0.0 {
		arg2.Amount.Float64 = arg.Amount
		arg2.Amount.Valid = true
	}
	if arg.Balance != 0.0 {
		arg2.Balance.Float64 = arg.Balance
		arg2.Balance.Valid = true
	}
	if arg.ContextID != 0 {
		arg2.ContextID.Int64 = arg.ContextID
		arg2.ContextID.Valid = true
	}
	if arg.ContextIDType != "" {
		arg2.ContextIDType.String = arg.ContextIDType
		arg2.ContextIDType.Valid = true
	}
	if arg.FirstPartyID != 0 {
		arg2.FirstPartyID.Int64 = int64(arg.FirstPartyID)
		arg2.FirstPartyID.Valid = true
	}
	if arg.Reason != "" {
		arg2.Reason.String = arg.Reason
		arg2.Reason.Valid = true
	}
	if arg.SecondPartyID != 0 {
		arg2.SecondPartyID.Int64 = int64(arg.SecondPartyID)
		arg2.SecondPartyID.Valid = true
	}
	if arg.Tax != 0.0 {
		arg2.Tax.Float64 = arg.Tax
		arg2.Tax.Valid = true
	}
	if arg.TaxReceiverID != 0 {
		arg2.TaxReceiverID.Int64 = int64(arg.TaxReceiverID)
		arg2.TaxReceiverID.Valid = true
	}
	err := r.q.CreateWalletJournalEntries(ctx, arg2)
	return err
}

func (r *Storage) GetWalletJournalEntry(ctx context.Context, characterID int32, entryID int64) (*model.WalletJournalEntry, error) {
	arg := queries.GetWalletJournalEntryParams{
		MyCharacterID: int64(characterID),
		ID:            entryID,
	}
	row, err := r.q.GetWalletJournalEntry(ctx, arg)
	if err != nil {
		return nil, err
	}
	return walletJournalEntryFromDBModel(row.WalletJournalEntry, row.WalletJournalEntryFirstParty, row.WalletJournalEntrySecondParty, row.WalletJournalEntryTaxReceiver), err
}

func walletJournalEntryFromDBModel(e queries.WalletJournalEntry, firstParty queries.WalletJournalEntryFirstParty, secondParty queries.WalletJournalEntrySecondParty, taxReceiver queries.WalletJournalEntryTaxReceiver) *model.WalletJournalEntry {
	e2 := &model.WalletJournalEntry{
		Date:          e.Date,
		Description:   e.Description,
		FirstParty:    eveEntityFromNullableDBModel(nullEveEntry(firstParty)),
		ID:            e.ID,
		MyCharacterID: int32(e.MyCharacterID),
		RefType:       e.RefType,
		SecondParty:   eveEntityFromNullableDBModel(nullEveEntry(secondParty)),
		TaxReceiver:   eveEntityFromNullableDBModel(nullEveEntry(taxReceiver)),
	}
	if e.Amount.Valid {
		e2.Amount = e.Amount.Float64
	}
	if e.Balance.Valid {
		e2.Balance = e.Balance.Float64
	}
	if e.ContextID.Valid {
		e2.ContextID = e.ContextID.Int64
	}
	if e.ContextIDType.Valid {
		e2.ContextIDType = e.ContextIDType.String
	}
	if e.Reason.Valid {
		e2.Reason = e.Reason.String
	}
	if e.Tax.Valid {
		e2.Tax = e.Tax.Float64
	}
	return e2
}
