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
		Amount:        arg.Amount,
		Balance:       arg.Balance,
		ContextID:     arg.ContextID,
		ContextIDType: arg.ContextIDType,
		Date:          arg.Date,
		Description:   arg.Description,
		ID:            arg.ID,
		MyCharacterID: int64(arg.MyCharacterID),
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

func (r *Storage) ListWalletJournalEntryIDs(ctx context.Context, characterID int32) ([]int64, error) {
	return r.q.ListWalletJournalEntryIDs(ctx, int64(characterID))
}

func (r *Storage) ListWalletJournalEntries(ctx context.Context, characterID int32) ([]*model.WalletJournalEntry, error) {
	rows, err := r.q.ListWalletJournalEntries(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	ee := make([]*model.WalletJournalEntry, len(rows))
	for i, row := range rows {
		ee[i] = walletJournalEntryFromDBModel(row.WalletJournalEntry, row.WalletJournalEntryFirstParty, row.WalletJournalEntrySecondParty, row.WalletJournalEntryTaxReceiver)
	}
	return ee, nil
}

func walletJournalEntryFromDBModel(e queries.WalletJournalEntry, firstParty queries.WalletJournalEntryFirstParty, secondParty queries.WalletJournalEntrySecondParty, taxReceiver queries.WalletJournalEntryTaxReceiver) *model.WalletJournalEntry {
	e2 := &model.WalletJournalEntry{
		Amount:        e.Amount,
		Balance:       e.Balance,
		ContextID:     e.ContextID,
		ContextIDType: e.ContextIDType,
		Date:          e.Date,
		Description:   e.Description,
		FirstParty:    eveEntityFromNullableDBModel(nullEveEntry(firstParty)),
		ID:            e.ID,
		MyCharacterID: int32(e.MyCharacterID),
		Reason:        e.Reason,
		RefType:       e.RefType,
		SecondParty:   eveEntityFromNullableDBModel(nullEveEntry(secondParty)),
		Tax:           e.Tax,
		TaxReceiver:   eveEntityFromNullableDBModel(nullEveEntry(taxReceiver)),
	}
	return e2
}
