package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) GetCorporationWalletBalance(ctx context.Context, arg CorporationDivision) (*app.CorporationWalletBalance, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCorporationWalletBalance %+v: %w", arg, err)
	}
	if arg.IsInvalid() {
		return nil, wrapErr(app.ErrInvalid)
	}
	o, err := st.qRO.GetCorporationWalletBalance(ctx, queries.GetCorporationWalletBalanceParams{
		CorporationID: int64(arg.CorporationID),
		DivisionID:    int64(arg.DivisionID),
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	return corporationWalletBalanceFromDBModel(o), nil
}

func (st *Storage) ListCorporationWalletBalances(ctx context.Context, corporationID int32) ([]*app.CorporationWalletBalance, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationWalletBalances for id %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCorporationWalletBalances(ctx, int64(corporationID))
	if err != nil {
		return nil, wrapErr(err)
	}
	ee := make([]*app.CorporationWalletBalance, len(rows))
	for i, r := range rows {
		ee[i] = corporationWalletBalanceFromDBModel(r)
	}
	return ee, nil
}

type UpdateOrCreateCorporationWalletBalanceParams struct {
	CorporationID int32
	DivisionID    int32
	Balance       float64
}

func (st *Storage) UpdateOrCreateCorporationWalletBalance(ctx context.Context, arg UpdateOrCreateCorporationWalletBalanceParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCorporationWalletBalance %+v: %w", arg, err)
	}
	if arg.CorporationID == 0 || arg.DivisionID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateCorporationWalletBalance(ctx, queries.UpdateOrCreateCorporationWalletBalanceParams{
		CorporationID: int64(arg.CorporationID),
		DivisionID:    int64(arg.DivisionID),
		Balance:       arg.Balance,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func corporationWalletBalanceFromDBModel(o queries.CorporationWalletBalance) *app.CorporationWalletBalance {
	o2 := &app.CorporationWalletBalance{
		CorporationID: int32(o.CorporationID),
		DivisionID:    int32(o.DivisionID),
		Balance:       o.Balance,
	}
	return o2
}
