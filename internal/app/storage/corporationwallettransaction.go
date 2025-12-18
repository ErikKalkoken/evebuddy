package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateCorporationWalletTransactionParams struct {
	ClientID      int32
	Date          time.Time
	DivisionID    int32
	EveTypeID     int32
	IsBuy         bool
	JournalRefID  int64
	LocationID    int64
	CorporationID int32
	Quantity      int32
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
		ClientID:      int64(arg.ClientID),
		Date:          arg.Date,
		DivisionID:    int64(arg.DivisionID),
		EveTypeID:     int64(arg.EveTypeID),
		IsBuy:         arg.IsBuy,
		JournalRefID:  arg.JournalRefID,
		LocationID:    arg.LocationID,
		CorporationID: int64(arg.CorporationID),
		Quantity:      int64(arg.Quantity),
		TransactionID: arg.TransactionID,
		UnitPrice:     arg.UnitPrice,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeleteCorporationWalletTransactions(ctx context.Context, corporationID int32, d app.Division) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCorporationWalletTransactions: %d %d %w", corporationID, d, err)
	}
	if corporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.DeleteCorporationWalletTransactions(ctx, queries.DeleteCorporationWalletTransactionsParams{
		CorporationID: int64(corporationID),
		DivisionID:    int64(d.ID()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Wallet transactions deleted for corporation", "corporationID", corporationID)
	return nil
}

type GetCorporationWalletTransactionParams struct {
	CorporationID int32
	DivisionID    int32
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
		CorporationID: int64(arg.CorporationID),
		DivisionID:    int64(arg.DivisionID),
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
		CorporationID: int64(arg.CorporationID),
		DivisionID:    int64(arg.DivisionID),
	})
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListCorporationWalletTransactions(ctx context.Context, arg CorporationDivision) ([]*app.CorporationWalletTransaction, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationWalletTransactions %+v: %w", arg, err)
	}
	if arg.IsInvalid() {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCorporationWalletTransactions(ctx, queries.ListCorporationWalletTransactionsParams{
		CorporationID: int64(arg.CorporationID),
		DivisionID:    int64(arg.DivisionID),
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
		DivisionID:   int32(o.DivisionID),
		Type:         eveTypeFromDBModel(et, eg, ec),
		ID:           o.ID,
		IsBuy:        o.IsBuy,
		JournalRefID: o.JournalRefID,
		Location: &app.EveLocationShort{
			ID:             o.LocationID,
			Name:           optional.New(locationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(systemSecurityStatus)},
		CorporationID: int32(o.CorporationID),
		Quantity:      int32(o.Quantity),
		TransactionID: o.TransactionID,
		UnitPrice:     o.UnitPrice,
	}
	if regionID.Valid && regionName.Valid {
		o2.Region = &app.EntityShort[int32]{
			ID:   int32(regionID.Int64),
			Name: regionName.String,
		}
	}
	return o2
}
