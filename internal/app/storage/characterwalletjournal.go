package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// FIXME: Wrong unique clause and missing index for ref_id

type CreateCharacterWalletJournalEntryParams struct {
	Amount        optional.Optional[float64]
	Balance       optional.Optional[float64]
	ContextID     optional.Optional[int64]
	ContextIDType optional.Optional[string]
	Date          time.Time
	Description   string
	FirstPartyID  optional.Optional[int64]
	RefID         int64
	CharacterID   int64
	Reason        optional.Optional[string]
	RefType       string
	SecondPartyID optional.Optional[int64]
	Tax           optional.Optional[float64]
	TaxReceiverID optional.Optional[int64]
}

func (st *Storage) CreateCharacterWalletJournalEntry(ctx context.Context, arg CreateCharacterWalletJournalEntryParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCharacterWalletJournalEntry: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.RefID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.CreateCharacterWalletJournalEntry(ctx, queries.CreateCharacterWalletJournalEntryParams{
		Amount:        arg.Amount.ValueOrZero(),
		Balance:       arg.Balance.ValueOrZero(),
		CharacterID:  arg.CharacterID,
		ContextID:     arg.ContextID.ValueOrZero(),
		ContextIDType: arg.ContextIDType.ValueOrZero(),
		Date:          arg.Date,
		Description:   arg.Description,
		FirstPartyID:  optional.ToNullInt64(arg.FirstPartyID),
		Reason:        arg.Reason.ValueOrZero(),
		RefID:         arg.RefID,
		RefType:       arg.RefType,
		SecondPartyID: optional.ToNullInt64(arg.SecondPartyID),
		Tax:           arg.Tax.ValueOrZero(),
		TaxReceiverID: optional.ToNullInt64(arg.TaxReceiverID),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

type GetCharacterWalletJournalEntryParams struct {
	CharacterID int64
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
		CharacterID:arg.CharacterID,
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

func (st *Storage) ListCharacterWalletJournalEntryIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("list wallet journal entry ids for character %d: %w", characterID, err)
	}
	if characterID == 0 {
		return set.Set[int64]{}, wrapErr(app.ErrInvalid)
	}
	ids, err := st.qRO.ListCharacterWalletJournalEntryRefIDs(ctx,characterID)
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListCharacterWalletJournalEntries(ctx context.Context, characterID int64) ([]*app.CharacterWalletJournalEntry, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("list wallet journal entries for character %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterWalletJournalEntries(ctx,characterID)
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
		Amount:        optional.FromZeroValue(o.Amount),
		Balance:       optional.FromZeroValue(o.Balance),
		ContextID:     optional.FromZeroValue(o.ContextID),
		ContextIDType: optional.FromZeroValue(o.ContextIDType),
		Date:          o.Date,
		Description:   o.Description,
		FirstParty:    eveEntityFromNullableDBModel(firstParty),
		ID:            o.ID,
		RefID:         o.RefID,
		CharacterID:  o.CharacterID,
		Reason:        optional.FromZeroValue(o.Reason),
		RefType:       o.RefType,
		SecondParty:   eveEntityFromNullableDBModel(secondParty),
		Tax:           optional.FromZeroValue(o.Tax),
		TaxReceiver:   eveEntityFromNullableDBModel(taxReceiver),
	}
	return o2
}
