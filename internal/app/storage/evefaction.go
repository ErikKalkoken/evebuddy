package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateEveFactionParams struct {
	ID                   int64
	CorporationID        optional.Optional[int64]
	Description          string
	IsUnique             bool
	MilitiaCorporationID optional.Optional[int64]
	Name                 string
	SizeFactor           float64
	SolarSystemID        optional.Optional[int64]
	StationCount         int64
	StationSystemCount   int64
}

func (st *Storage) CreateEveFaction(ctx context.Context, arg CreateEveFactionParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateEveFaction: %+v, %w", arg, err)
	}
	if arg.ID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.CreateEveFaction(ctx, queries.CreateEveFactionParams{
		ID:                   arg.ID,
		Description:          arg.Description,
		Name:                 arg.Name,
		CorporationID:        optional.ToNullInt64(arg.CorporationID),
		IsUnique:             arg.IsUnique,
		MilitiaCorporationID: optional.ToNullInt64(arg.MilitiaCorporationID),
		SizeFactor:           arg.SizeFactor,
		SolarSystemID:        optional.ToNullInt64(arg.SolarSystemID),
		StationCount:         arg.StationCount,
		StationSystemCount:   arg.StationSystemCount,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetEveFaction(ctx context.Context, id int64) (*app.EveFaction, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetEveFaction: %d, %w", id, err)
	}
	if id == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetEveFaction(ctx, id)
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	cc := nullCorporation{
		id:       r.EveFaction.CorporationID,
		category: NewNullString(eveEntityCorporation),
		name:     r.CorporationName,
	}
	mc := nullMilitiaCorporation{
		id:       r.EveFaction.MilitiaCorporationID,
		category: NewNullString(eveEntityCorporation),
		name:     r.MilitiaCorporationName,
	}
	es := nullEntity{
		id:   r.EveFaction.SolarSystemID,
		name: r.SolarSystemName,
	}
	o := eveFactionFromDBModel(r.EveFaction, cc, mc, es)
	return o, nil
}

func eveFactionFromDBModel(ef queries.EveFaction, cc nullCorporation, mc nullMilitiaCorporation, es nullEntity) *app.EveFaction {
	return &app.EveFaction{
		ID:                 ef.ID,
		Corporation:        eveEntityFromNullableDBModel(nullEveEntity(cc)),
		Description:        ef.Description,
		IsUnique:           ef.IsUnique,
		MilitiaCorporation: eveEntityFromNullableDBModel(nullEveEntity(mc)),
		Name:               ef.Name,
		SizeFactor:         ef.SizeFactor,
		SolarSystem:        entityShortFromNullableDBModel(es),
		StationCount:       ef.StationCount,
		StationSystemCount: ef.StationSystemCount,
	}
}
