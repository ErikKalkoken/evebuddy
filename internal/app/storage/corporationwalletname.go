package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) GetCorporationWalletName(ctx context.Context, arg CorporationDivision) (*app.CorporationWalletName, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCorporationWalletName %+v: %w", arg, err)
	}
	if arg.IsInvalid() {
		return nil, wrapErr(app.ErrInvalid)
	}
	o, err := st.qRO.GetCorporationWalletName(ctx, queries.GetCorporationWalletNameParams{
		CorporationID: int64(arg.CorporationID),
		DivisionID:    int64(arg.DivisionID),
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	return corporationWalletNameFromDBModel(o), nil
}

func (st *Storage) ListCorporationWalletNames(ctx context.Context, corporationID int32) ([]*app.CorporationWalletName, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationWalletNames for id %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCorporationWalletNames(ctx, int64(corporationID))
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]*app.CorporationWalletName, len(rows))
	for i, r := range rows {
		oo[i] = corporationWalletNameFromDBModel(r)
	}
	return oo, nil
}

type UpdateOrCreateCorporationWalletNameParams struct {
	CorporationID int32
	DivisionID    int32
	Name          string
}

func (st *Storage) UpdateOrCreateCorporationWalletName(ctx context.Context, arg UpdateOrCreateCorporationWalletNameParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCorporationWalletName %+v: %w", arg, err)
	}
	if arg.CorporationID == 0 || arg.DivisionID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateCorporationWalletName(ctx, queries.UpdateOrCreateCorporationWalletNameParams{
		CorporationID: int64(arg.CorporationID),
		DivisionID:    int64(arg.DivisionID),
		Name:          arg.Name,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func corporationWalletNameFromDBModel(o queries.CorporationWalletName) *app.CorporationWalletName {
	o2 := &app.CorporationWalletName{
		CorporationID: int32(o.CorporationID),
		DivisionID:    int32(o.DivisionID),
		Name:          o.Name,
	}
	return o2
}
