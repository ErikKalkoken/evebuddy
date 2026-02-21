package storage

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateCorporationWalletJournalEntryParams struct {
	Amount        optional.Optional[float64]
	Balance       optional.Optional[float64]
	ContextID     optional.Optional[int64]
	ContextIDType optional.Optional[string]
	Date          time.Time
	Description   string
	DivisionID    int64
	FirstPartyID  optional.Optional[int64]
	RefID         int64
	CorporationID int64
	Reason        optional.Optional[string]
	RefType       string
	SecondPartyID optional.Optional[int64]
	Tax           optional.Optional[float64]
	TaxReceiverID optional.Optional[int64]
}

func (st *Storage) CreateCorporationWalletJournalEntry(ctx context.Context, arg CreateCorporationWalletJournalEntryParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCorporationWalletJournalEntry: %+v: %w", arg, err)
	}
	if arg.CorporationID == 0 || arg.DivisionID == 0 || arg.RefID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.CreateCorporationWalletJournalEntry(ctx, queries.CreateCorporationWalletJournalEntryParams{
		Amount:        arg.Amount.ValueOrZero(),
		Balance:       arg.Balance.ValueOrZero(),
		ContextID:     arg.ContextID.ValueOrZero(),
		ContextIDType: arg.ContextIDType.ValueOrZero(),
		Date:          arg.Date,
		Description:   arg.Description,
		DivisionID:    arg.DivisionID,
		RefID:         arg.RefID,
		CorporationID: arg.CorporationID,
		RefType:       arg.RefType,
		Reason:        arg.Reason.ValueOrZero(),
		Tax:           arg.Tax.ValueOrZero(),
		FirstPartyID:  optional.ToNullInt64(arg.FirstPartyID),
		SecondPartyID: optional.ToNullInt64(arg.SecondPartyID),
		TaxReceiverID: optional.ToNullInt64(arg.TaxReceiverID),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeleteCorporationWalletJournal(ctx context.Context, corporationID int64, d app.Division) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCorporationWalletJournalEntries: %d %d: %w", corporationID, d, err)
	}
	if corporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.DeleteCorporationWalletJournalEntries(ctx, queries.DeleteCorporationWalletJournalEntriesParams{
		CorporationID: corporationID,
		DivisionID:    d.ID(),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Wallet journal deleted for corporation", "corporationID", corporationID)
	return nil
}

type GetCorporationWalletJournalEntryParams struct {
	CorporationID int64
	DivisionID    int64
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
		CorporationID: arg.CorporationID,
		DivisionID:    arg.DivisionID,
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
	CorporationID int64
	DivisionID    int64
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
		CorporationID: arg.CorporationID,
		DivisionID:    arg.DivisionID,
	})
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) ListCorporationWalletJournalEntries(ctx context.Context, arg CorporationDivision) ([]*app.CorporationWalletJournalEntry, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationWalletJournalEntries: %+v: %w", arg, err)
	}
	if arg.IsInvalid() {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCorporationWalletJournalEntries(ctx, queries.ListCorporationWalletJournalEntriesParams{
		CorporationID: arg.CorporationID,
		DivisionID:    arg.DivisionID,
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
		Amount:        optional.FromZeroValue(o.Amount),
		Balance:       optional.FromZeroValue(o.Balance),
		ContextID:     optional.FromZeroValue(o.ContextID),
		ContextIDType: optional.FromZeroValue(o.ContextIDType),
		Date:          o.Date,
		Description:   o.Description,
		DivisionID:    o.DivisionID,
		FirstParty:    eveEntityFromNullableDBModel(firstParty),
		ID:            o.ID,
		RefID:         o.RefID,
		CorporationID: o.CorporationID,
		Reason:        optional.FromZeroValue(o.Reason),
		RefType:       o.RefType,
		SecondParty:   eveEntityFromNullableDBModel(secondParty),
		Tax:           optional.FromZeroValue(o.Tax),
		TaxReceiver:   eveEntityFromNullableDBModel(taxReceiver),
	}
	return o2
}
