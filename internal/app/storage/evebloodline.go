package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateEveBloodlineParams struct {
	Charisma      optional.Optional[int64]
	CorporationID int64
	Description   string
	ID            int64
	Intelligence  optional.Optional[int64]
	Memory        optional.Optional[int64]
	Name          string
	Perception    optional.Optional[int64]
	RaceID        int64
	ShipTypeID    optional.Optional[int64]
	Willpower     optional.Optional[int64]
}

func (st *Storage) CreateEveBloodline(ctx context.Context, arg CreateEveBloodlineParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateEveBloodline: %+v, %w", arg, err)
	}
	if arg.ID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.CreateEveBloodline(ctx, queries.CreateEveBloodlineParams{
		ID:            arg.ID,
		Charisma:      optional.ToNullInt64(arg.Charisma),
		CorporationID: arg.CorporationID,
		Description:   arg.Description,
		Intelligence:  optional.ToNullInt64(arg.Intelligence),
		Memory:        optional.ToNullInt64(arg.Memory),
		Name:          arg.Name,
		Perception:    optional.ToNullInt64(arg.Perception),
		RaceID:        arg.RaceID,
		ShipTypeID:    optional.ToNullInt64(arg.ShipTypeID),
		Willpower:     optional.ToNullInt64(arg.Willpower),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetEveBloodline(ctx context.Context, id int64) (*app.EveBloodline, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetEveBloodline: %d, %w", id, err)
	}
	if id == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetEveBloodline(ctx, id)
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	eea := nullEveEntity{
		id:       r.EveRace.FactionID,
		category: NewNullString(eveEntityFaction),
		name:     r.FactionName,
	}
	o := eveBloodlineFromDBModel(r.EveBloodline, r.EveEntity, r.EveRace, eea)
	return o, nil
}

func eveBloodlineFromDBModel(
	eb queries.EveBloodline,
	eec queries.EveEntity,
	er queries.EveRace,
	eea nullEveEntity,
) *app.EveBloodline {
	return &app.EveBloodline{
		Charisma:     optional.FromNullInt64(eb.Charisma),
		Corporation:  eveEntityFromDBModel(eec),
		Description:  eb.Description,
		ID:           eb.ID,
		Intelligence: optional.FromNullInt64(eb.Intelligence),
		Memory:       optional.FromNullInt64(eb.Memory),
		Name:         eb.Name,
		Perception:   optional.FromNullInt64(eb.Perception),
		Race:         eveRaceFromDBModel(er, eea),
		ShipTypeID:   optional.FromNullInt64(eb.ShipTypeID),
		Willpower:    optional.FromNullInt64(eb.Willpower),
	}
}
