package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateEveCharacterParams struct {
	ID             int64
	AllianceID     optional.Optional[int64]
	Birthday       time.Time
	CorporationID  int64
	Description    optional.Optional[string]
	FactionID      optional.Optional[int64]
	Gender         string
	Name           string
	RaceID         int64
	SecurityStatus optional.Optional[float64]
	Title          optional.Optional[string]
}

func (st *Storage) UpdateOrCreateEveCharacter(ctx context.Context, arg CreateEveCharacterParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateEveCharacter: %+v: %w", arg, err)
	}
	if arg.ID == 0 || arg.CorporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateEveCharacter(ctx, queries.UpdateOrCreateEveCharacterParams{
		ID:             arg.ID,
		Birthday:       arg.Birthday,
		CorporationID:  arg.CorporationID,
		Description:    arg.Description.ValueOrZero(),
		Gender:         arg.Gender,
		Name:           arg.Name,
		RaceID:         arg.RaceID,
		SecurityStatus: arg.SecurityStatus.ValueOrZero(),
		Title:          arg.Title.ValueOrZero(),
		AllianceID:     optional.ToNullInt64(arg.AllianceID),
		FactionID:      optional.ToNullInt64(arg.FactionID),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeleteEveCharacter(ctx context.Context, characterID int64) error {
	err := st.qRW.DeleteEveCharacter(ctx, characterID)
	if err != nil {
		return fmt.Errorf("delete EveCharacter %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) GetEveCharacter(ctx context.Context, characterID int64) (*app.EveCharacter, error) {
	r, err := st.qRO.GetEveCharacter(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("get EveCharacter %d: %w", characterID, convertGetError(err))
	}
	alliance := nullEveEntry{
		id:       r.EveCharacter.AllianceID,
		name:     r.AllianceName,
		category: r.AllianceCategory,
	}
	faction := nullEveEntry{
		id:       r.EveCharacter.FactionID,
		name:     r.FactionName,
		category: r.FactionCategory,
	}
	c := eveCharacterFromDBModel(
		r.EveCharacter,
		r.EveEntity,
		r.EveRace,
		alliance,
		faction,
	)
	return c, nil
}

func (st *Storage) ListEveCharacterIDs(ctx context.Context) (set.Set[int64], error) {
	ids, err := st.qRO.ListEveCharacterIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list EveCharacterIDs: %w", err)
	}
	ids2 := set.Of(ids...)
	return ids2, nil
}

func (st *Storage) UpdateEveCharacter(ctx context.Context, c *app.EveCharacter) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateEveCharacter: %+v: %w", c, err)
	}
	if c.ID == 0 || c.Corporation == nil {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateEveCharacter(ctx, queries.UpdateEveCharacterParams{
		ID:             c.ID,
		CorporationID:  c.Corporation.ID,
		Description:    c.Description.ValueOrZero(),
		Name:           c.Name,
		SecurityStatus: c.SecurityStatus.ValueOrZero(),
		Title:          c.Title.ValueOrZero(),
		AllianceID:     NewNullInt64(c.AllianceID()),
		FactionID:      NewNullInt64(c.FactionID()),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) UpdateEveCharacterName(ctx context.Context, characterID int64, name string) error {
	if characterID == 0 || name == "" {
		return fmt.Errorf("UpdateEveCharacterName: %w", app.ErrInvalid)
	}
	if err := st.qRW.UpdateEveCharacterName(ctx, queries.UpdateEveCharacterNameParams{
		ID:   characterID,
		Name: name,
	}); err != nil {
		return fmt.Errorf("UpdateEveCharacterName %d: %w", characterID, err)
	}
	return nil
}

func eveCharacterFromDBModel(
	character queries.EveCharacter,
	corporation queries.EveEntity,
	race queries.EveRace,
	alliance nullEveEntry,
	faction nullEveEntry,
) *app.EveCharacter {
	o := app.EveCharacter{
		Alliance:       eveEntityFromNullableDBModel(alliance),
		Birthday:       character.Birthday,
		Corporation:    eveEntityFromDBModel(corporation),
		Description:    optional.FromZeroValue(character.Description),
		Gender:         character.Gender,
		Faction:        eveEntityFromNullableDBModel(faction),
		ID:             character.ID,
		Name:           character.Name,
		Race:           eveRaceFromDBModel(race),
		SecurityStatus: optional.FromZeroValue(character.SecurityStatus),
		Title:          optional.FromZeroValue(character.Title),
	}
	return &o
}
