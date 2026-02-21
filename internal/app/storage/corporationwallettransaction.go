package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateCorporationWalletTransactionParams struct {
	ClientID      int64
	Date          time.Time
	DivisionID    int64
	EveTypeID     int64
	IsBuy         bool
	JournalRefID  int64
	LocationID    int64
	CorporationID int64
	Quantity      int64
	TransactionID int64
	UnitPrice     float64
}

func (st *Storage) CreateCorporationWalletTransaction(ctx context.Context, arg CreateCorporationWalletTransactionParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateCorporationWalletTransaction: %+v: %w", arg, err)
	}
	if arg.CorporationID == 0 || arg.DivisionID == 0 || arg.EveTypeID == 0 || arg.ClientID == 0 || arg.JournalRefID == 0 || arg.LocationID == 0 || arg.TransactionID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.CreateCorporationWalletTransaction(ctx, queries.CreateCorporationWalletTransactionParams{
		ClientID:     arg.ClientID,
		Date:          arg.Date,
		DivisionID:   arg.DivisionID,
		EveTypeID:    arg.EveTypeID,
		IsBuy:         arg.IsBuy,
		JournalRefID:  arg.JournalRefID,
		LocationID:    arg.LocationID,
		CorporationID:arg.CorporationID,
		Quantity:     arg.Quantity,
		TransactionID: arg.TransactionID,
		UnitPrice:     arg.UnitPrice,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeleteCorporationWalletTransactions(ctx context.Context, corporationID int64, d app.Division) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCorporationWalletTransactions: %d %d %w", corporationID, d, err)
	}
	if corporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.DeleteCorporationWalletTransactions(ctx, queries.DeleteCorporationWalletTransactionsParams{
		CorporationID:corporationID,
		DivisionID:   d.ID(),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Wallet transactions deleted for corporation", "corporationID", corporationID)
	return nil
}

type GetCorporationWalletTransactionParams struct {
	CorporationID int64
	DivisionID    int64
	TransactionID int64
}

func (st *Storage) GetCorporationWalletTransaction(ctx context.Context, arg GetCorporationWalletTransactionParams) (*app.CorporationWalletTransaction, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCorporationWalletTransaction: %+v: %w", arg, err)
	}
	if arg.CorporationID == 0 || arg.DivisionID == 0 || arg.TransactionID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCorporationWalletTransaction(ctx, queries.GetCorporationWalletTransactionParams{
		CorporationID:arg.CorporationID,
		DivisionID:   arg.DivisionID,
		TransactionID: arg.TransactionID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	o := corporationWalletTransactionFromDBModel(
		r.CorporationWalletTransaction,
		r.EveEntity,
		r.EveType,
		r.EveGroup,
		r.EveCategory,
		r.LocationName,
		r.SystemSecurityStatus,
		r.RegionID,
		r.RegionName,
	)
	return o, err
}

func (st *Storage) ListCorporationWalletTransactionIDs(ctx context.Context, arg CorporationDivision) (set.Set[int64], error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationWalletTransactionIDs %+v: %w", arg, err)
	}
	if arg.IsInvalid() {
		return set.Set[int64]{}, wrapErr(app.ErrInvalid)
	}
	ids, err := st.qRO.ListCorporationWalletTransactionIDs(ctx, queries.ListCorporationWalletTransactionIDsParams{
		CorporationID:arg.CorporationID,
		DivisionID:   arg.DivisionID,
	})
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) ListCorporationWalletTransactions(ctx context.Context, arg CorporationDivision) ([]*app.CorporationWalletTransaction, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationWalletTransactions %+v: %w", arg, err)
	}
	if arg.IsInvalid() {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCorporationWalletTransactions(ctx, queries.ListCorporationWalletTransactionsParams{
		CorporationID:arg.CorporationID,
		DivisionID:   arg.DivisionID,
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]*app.CorporationWalletTransaction, len(rows))
	for i, r := range rows {
		oo[i] = corporationWalletTransactionFromDBModel(
			r.CorporationWalletTransaction,
			r.EveEntity,
			r.EveType,
			r.EveGroup,
			r.EveCategory,
			r.LocationName,
			r.SystemSecurityStatus,
			r.RegionID,
			r.RegionName,
		)
	}
	return oo, nil
}

func corporationWalletTransactionFromDBModel(
	o queries.CorporationWalletTransaction,
	client queries.EveEntity,
	et queries.EveType,
	eg queries.EveGroup,
	ec queries.EveCategory,
	locationName string,
	systemSecurityStatus sql.NullFloat64,
	regionID sql.NullInt64,
	regionName sql.NullString,
) *app.CorporationWalletTransaction {
	o2 := &app.CorporationWalletTransaction{
		Client:       eveEntityFromDBModel(client),
		Date:         o.Date,
		DivisionID:  o.DivisionID,
		Type:         eveTypeFromDBModel(et, eg, ec),
		ID:           o.ID,
		IsBuy:        o.IsBuy,
		JournalRefID: o.JournalRefID,
		Location: &app.EveLocationShort{
			ID:             o.LocationID,
			Name:           optional.New(locationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(systemSecurityStatus)},
		CorporationID:o.CorporationID,
		Quantity:     o.Quantity,
		TransactionID: o.TransactionID,
		UnitPrice:     o.UnitPrice,
	}
	if regionID.Valid && regionName.Valid {
		o2.Region = &app.EntityShort{
			ID:  regionID.Int64,
			Name: regionName.String,
		}
	}
	return o2
}
