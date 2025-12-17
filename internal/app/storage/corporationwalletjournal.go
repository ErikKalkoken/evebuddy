package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/kx/set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateCorporationWalletJournalEntryParams struct {
	Amount        float64
	Balance       float64
	ContextID     int64
	ContextIDType string
	Date          time.Time
	Description   string
	DivisionID    int32
	FirstPartyID  int32
	RefID         int64
	CorporationID int32
	Reason        string
	RefType       string
	SecondPartyID int32
	Tax           float64
	TaxReceiverID int32
}

func (st *Storage) CreateCorporationWalletJournalEntry(ctx context.Context, arg CreateCorporationWalletJournalEntryParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCorporationWalletJournalEntry: %+v: %w", arg, err)
	}
	if arg.CorporationID == 0 || arg.DivisionID == 0 || arg.RefID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	arg2 := queries.CreateCorporationWalletJournalEntryParams{
		Amount:        arg.Amount,
		Balance:       arg.Balance,
		ContextID:     arg.ContextID,
		ContextIDType: arg.ContextIDType,
		Date:          arg.Date,
		Description:   arg.Description,
		DivisionID:    int64(arg.DivisionID),
		RefID:         arg.RefID,
		CorporationID: int64(arg.CorporationID),
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
	err := st.qRW.CreateCorporationWalletJournalEntry(ctx, arg2)
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeleteCorporationWalletJournal(ctx context.Context, corporationID int32, d app.Division) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCorporationWalletJournalEntries: %d %d: %w", corporationID, d, err)
	}
	if corporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.DeleteCorporationWalletJournalEntries(ctx, queries.DeleteCorporationWalletJournalEntriesParams{
		CorporationID: int64(corporationID),
		DivisionID:    int64(d.ID()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Wallet journal deleted for corporation", "corporationID", corporationID)
	return nil
}

type GetCorporationWalletJournalEntryParams struct {
	CorporationID int32
	DivisionID    int32
	RefID         int64
}

func (st *Storage) GetCorporationWalletJournalEntry(ctx context.Context, arg GetCorporationWalletJournalEntryParams) (*app.CorporationWalletJournalEntry, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCorporationWalletJournalEntry: %+v: %w", arg, err)
	}
	if arg.CorporationID == 0 || arg.DivisionID == 0 || arg.RefID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCorporationWalletJournalEntry(ctx, queries.GetCorporationWalletJournalEntryParams{
		CorporationID: int64(arg.CorporationID),
		DivisionID:    int64(arg.DivisionID),
		RefID:         arg.RefID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	o := r.CorporationWalletJournalEntry
	firstParty := nullEveEntry{id: o.FirstPartyID, name: r.FirstName, category: r.FirstCategory}
	secondParty := nullEveEntry{id: o.SecondPartyID, name: r.SecondName, category: r.SecondCategory}
	taxReceiver := nullEveEntry{id: o.TaxReceiverID, name: r.TaxName, category: r.TaxCategory}
	return corporationWalletJournalEntryFromDBModel(o, firstParty, secondParty, taxReceiver), err
}

type CorporationDivision struct {
	CorporationID int32
	DivisionID    int32
}

func (cd CorporationDivision) IsInvalid() bool {
	return cd.CorporationID <= 0 || cd.DivisionID <= 0
}

func (st *Storage) ListCorporationWalletJournalEntryIDs(ctx context.Context, arg CorporationDivision) (set.Set[int64], error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationWalletJournalEntryIDs: %+v: %w", arg, err)
	}
	if arg.IsInvalid() {
		return set.Set[int64]{}, wrapErr(app.ErrInvalid)
	}
	ids, err := st.qRO.ListCorporationWalletJournalEntryRefIDs(ctx, queries.ListCorporationWalletJournalEntryRefIDsParams{
		CorporationID: int64(arg.CorporationID),
		DivisionID:    int64(arg.DivisionID),
	})
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListCorporationWalletJournalEntries(ctx context.Context, arg CorporationDivision) ([]*app.CorporationWalletJournalEntry, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationWalletJournalEntries: %+v: %w", arg, err)
	}
	if arg.IsInvalid() {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCorporationWalletJournalEntries(ctx, queries.ListCorporationWalletJournalEntriesParams{
		CorporationID: int64(arg.CorporationID),
		DivisionID:    int64(arg.DivisionID),
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	ee := make([]*app.CorporationWalletJournalEntry, len(rows))
	for i, r := range rows {
		o := r.CorporationWalletJournalEntry
		firstParty := nullEveEntry{id: o.FirstPartyID, name: r.FirstName, category: r.FirstCategory}
		secondParty := nullEveEntry{id: o.SecondPartyID, name: r.SecondName, category: r.SecondCategory}
		taxReceiver := nullEveEntry{id: o.TaxReceiverID, name: r.TaxName, category: r.TaxCategory}
		ee[i] = corporationWalletJournalEntryFromDBModel(o, firstParty, secondParty, taxReceiver)
	}
	return ee, nil
}

func corporationWalletJournalEntryFromDBModel(
	o queries.CorporationWalletJournalEntry,
	firstParty, secondParty, taxReceiver nullEveEntry,
) *app.CorporationWalletJournalEntry {
	o2 := &app.CorporationWalletJournalEntry{
		Amount:        o.Amount,
		Balance:       o.Balance,
		ContextID:     o.ContextID,
		ContextIDType: o.ContextIDType,
		Date:          o.Date,
		Description:   o.Description,
		DivisionID:    int32(o.DivisionID),
		FirstParty:    eveEntityFromNullableDBModel(firstParty),
		ID:            o.ID,
		RefID:         o.RefID,
		CorporationID: int32(o.CorporationID),
		Reason:        o.Reason,
		RefType:       o.RefType,
		SecondParty:   eveEntityFromNullableDBModel(secondParty),
		Tax:           o.Tax,
		TaxReceiver:   eveEntityFromNullableDBModel(taxReceiver),
	}
	return o2
}
