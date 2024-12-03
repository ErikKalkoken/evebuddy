package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreatePlanetPinContentParams struct {
	Amount      int
	EveTypeID   int32
	PlanetPinID int64
}

func (st *Storage) CreatePlanetPinContent(ctx context.Context, arg CreatePlanetPinContentParams) error {
	if arg.EveTypeID == 0 {
		return fmt.Errorf("invalid ID for planet pin content: %+v", arg)
	}
	arg2 := queries.CreatePlanetPinContentParams{
		Amount: int64(arg.Amount),
		TypeID: int64(arg.EveTypeID),
		PinID:  arg.PlanetPinID,
	}
	if err := st.q.CreatePlanetPinContent(ctx, arg2); err != nil {
		return fmt.Errorf("create planet pin content %v, %w", arg, err)
	}
	return nil
}

func (st *Storage) GetPlanetPinContent(ctx context.Context, pinID int64, typeID int32) (*app.PlanetPinContent, error) {
	arg := queries.GetPlanetPinContentParams{
		PinID:  pinID,
		TypeID: int64(typeID),
	}
	c, err := st.q.GetPlanetPinContent(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("get PlanetPinContent for %+v: %w", arg, err)
	}
	return planetPinContentFromDBModel(c), nil
}

func (st *Storage) ListPlanetPinContents(ctx context.Context, pinID int64) ([]*app.PlanetPinContent, error) {
	rows, err := st.q.ListPlanetPinContents(ctx, pinID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("list PlanetPinContent for ID %d: %w", pinID, err)
	}
	oo := make([]*app.PlanetPinContent, len(rows))
	for i, r := range rows {
		oo[i] = planetPinContentFromDBModel(queries.GetPlanetPinContentRow(r))
	}
	return oo, nil
}

func planetPinContentFromDBModel(r queries.GetPlanetPinContentRow) *app.PlanetPinContent {
	return &app.PlanetPinContent{
		Amount: int(r.PlanetPinContent.Amount),
		Type:   eveTypeFromDBModel(r.EveType, r.EveGroup, r.EveCategory),
	}
}
