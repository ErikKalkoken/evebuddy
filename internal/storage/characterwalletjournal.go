package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
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

func (r *Storage) CreateCharacterWalletJournalEntry(ctx context.Context, arg CreateCharacterWalletJournalEntryParams) error {
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
	err := r.q.CreateCharacterWalletJournalEntry(ctx, arg2)
	return err
}

func (r *Storage) GetCharacterWalletJournalEntry(ctx context.Context, characterID int32, refID int64) (*model.CharacterWalletJournalEntry, error) {
	arg := queries.GetCharacterWalletJournalEntryParams{
		CharacterID: int64(characterID),
		RefID:       refID,
	}
	row, err := r.q.GetCharacterWalletJournalEntry(ctx, arg)
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

func (r *Storage) ListCharacterWalletJournalEntryIDs(ctx context.Context, characterID int32) ([]int64, error) {
	return r.q.ListCharacterWalletJournalEntryRefIDs(ctx, int64(characterID))
}

func (r *Storage) ListCharacterWalletJournalEntries(ctx context.Context, characterID int32) ([]*model.CharacterWalletJournalEntry, error) {
	rows, err := r.q.ListCharacterWalletJournalEntries(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	ee := make([]*model.CharacterWalletJournalEntry, len(rows))
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
	e queries.CharacterWalletJournalEntry,
	firstParty queries.CharacterWalletJournalEntryFirstParty,
	secondParty queries.CharacterWalletJournalEntrySecondParty,
	taxReceiver queries.CharacterWalletJournalEntryTaxReceiver,
) *model.CharacterWalletJournalEntry {
	e2 := &model.CharacterWalletJournalEntry{
		Amount:        e.Amount,
		Balance:       e.Balance,
		ContextID:     e.ContextID,
		ContextIDType: e.ContextIDType,
		Date:          e.Date,
		Description:   e.Description,
		FirstParty:    eveEntityFromNullableDBModel(nullEveEntry(firstParty)),
		RefID:         e.ID,
		CharacterID:   int32(e.CharacterID),
		Reason:        e.Reason,
		RefType:       e.RefType,
		SecondParty:   eveEntityFromNullableDBModel(nullEveEntry(secondParty)),
		Tax:           e.Tax,
		TaxReceiver:   eveEntityFromNullableDBModel(nullEveEntry(taxReceiver)),
	}
	return e2
}
