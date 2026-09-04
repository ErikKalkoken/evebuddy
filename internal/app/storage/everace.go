package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type UpdateOrCreateEveRaceParams struct {
	Description string
	FactionID   optional.Optional[int64]
	ID          int64
	Name        string
}

func (st *Storage) UpdateOrCreateEveRace(ctx context.Context, arg UpdateOrCreateEveRaceParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateEveRace: %+v, %w", arg, err)
	}
	if arg.ID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateEveRace(ctx, queries.UpdateOrCreateEveRaceParams{
		Description: arg.Description,
		FactionID:   optional.ToNullInt64(arg.FactionID),
		ID:          arg.ID,
		Name:        arg.Name,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetEveRace(ctx context.Context, id int64) (*app.EveRace, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetEveRace: %d, %w", id, err)
	}
	if id == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetEveRace(ctx, id)
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	alliance := nullEveEntity{
		id:       r.EveRace.FactionID,
		category: NewNullString(eveEntityFaction),
		name:     r.AllianceName,
	}
	o := eveRaceFromDBModel(r.EveRace, alliance)
	return o, nil
}

func eveRaceFromDBModel(er queries.EveRace, alliance nullEveEntity) *app.EveRace {
	return &app.EveRace{
		Faction:     eveEntityFromNullableDBModel(alliance),
		Description: er.Description,
		ID:          er.ID,
		Name:        er.Name,
	}
}
