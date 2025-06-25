package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) GetCorporationHangarName(ctx context.Context, arg CorporationDivision) (*app.CorporationHangarName, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCorporationHangarName %+v: %w", arg, err)
	}
	if arg.IsInvalid() {
		return nil, wrapErr(app.ErrInvalid)
	}
	o, err := st.qRO.GetCorporationHangarName(ctx, queries.GetCorporationHangarNameParams{
		CorporationID: int64(arg.CorporationID),
		DivisionID:    int64(arg.DivisionID),
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	return corporationHangarNameFromDBModel(o), nil
}

func (st *Storage) ListCorporationHangarNames(ctx context.Context, corporationID int32) ([]*app.CorporationHangarName, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCorporationHangarNames for id %d: %w", corporationID, err)
	}
	if corporationID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCorporationHangarNames(ctx, int64(corporationID))
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]*app.CorporationHangarName, len(rows))
	for i, r := range rows {
		oo[i] = corporationHangarNameFromDBModel(r)
	}
	return oo, nil
}

type UpdateOrCreateCorporationHangarNameParams struct {
	CorporationID int32
	DivisionID    int32
	Name          string
}

func (st *Storage) UpdateOrCreateCorporationHangarName(ctx context.Context, arg UpdateOrCreateCorporationHangarNameParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCorporationHangarName %+v: %w", arg, err)
	}
	if arg.CorporationID == 0 || arg.DivisionID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateCorporationHangarName(ctx, queries.UpdateOrCreateCorporationHangarNameParams{
		CorporationID: int64(arg.CorporationID),
		DivisionID:    int64(arg.DivisionID),
		Name:          arg.Name,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func corporationHangarNameFromDBModel(o queries.CorporationHangarName) *app.CorporationHangarName {
	o2 := &app.CorporationHangarName{
		CorporationID: int32(o.CorporationID),
		DivisionID:    int32(o.DivisionID),
		Name:          o.Name,
	}
	return o2
}
