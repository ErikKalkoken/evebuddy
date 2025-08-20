package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type CreateEveCharacterParams struct {
	ID             int32
	AllianceID     int32
	Birthday       time.Time
	CorporationID  int32
	Description    string
	FactionID      int32
	Gender         string
	Name           string
	RaceID         int32
	SecurityStatus float64
	Title          string
}

func (st *Storage) CreateEveCharacter(ctx context.Context, arg CreateEveCharacterParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("CreateEveCharacter: %+v: %w", arg, err)
	}
	if arg.ID == 0 || arg.CorporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	arg2 := queries.CreateEveCharacterParams{
		ID:             int64(arg.ID),
		Birthday:       arg.Birthday,
		CorporationID:  int64(arg.CorporationID),
		Description:    arg.Description,
		Gender:         arg.Gender,
		Name:           arg.Name,
		RaceID:         int64(arg.RaceID),
		SecurityStatus: arg.SecurityStatus,
		Title:          arg.Title,
	}
	if arg.AllianceID != 0 {
		arg2.AllianceID.Int64 = int64(arg.AllianceID)
		arg2.AllianceID.Valid = true
	}
	if arg.FactionID != 0 {
		arg2.FactionID.Int64 = int64(arg.FactionID)
		arg2.FactionID.Valid = true
	}
	err := st.qRW.CreateEveCharacter(ctx, arg2)
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) DeleteEveCharacter(ctx context.Context, characterID int32) error {
	err := st.qRW.DeleteEveCharacter(ctx, int64(characterID))
	if err != nil {
		return fmt.Errorf("delete EveCharacter %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) GetEveCharacter(ctx context.Context, characterID int32) (*app.EveCharacter, error) {
	r, err := st.qRO.GetEveCharacter(ctx, int64(characterID))
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

func (st *Storage) ListEveCharacterIDs(ctx context.Context) (set.Set[int32], error) {
	ids, err := st.qRO.ListEveCharacterIDs(ctx)
	if err != nil {
		return set.Set[int32]{}, fmt.Errorf("list EveCharacterIDs: %w", err)
	}
	ids2 := set.Of(convertNumericSlice[int32](ids)...)
	return ids2, nil
}

func (st *Storage) UpdateEveCharacter(ctx context.Context, c *app.EveCharacter) error {
	arg := queries.UpdateEveCharacterParams{
		ID:             int64(c.ID),
		CorporationID:  int64(c.Corporation.ID),
		Description:    c.Description,
		Name:           c.Name,
		SecurityStatus: c.SecurityStatus,
		Title:          c.Title,
	}
	if c.HasAlliance() {
		arg.AllianceID.Int64 = int64(c.Alliance.ID)
		arg.AllianceID.Valid = true
	}
	if c.HasFaction() {
		arg.FactionID.Int64 = int64(c.Faction.ID)
		arg.FactionID.Valid = true
	}
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateEveCharacter: %+v: %w", arg, err)
	}
	if arg.ID == 0 || arg.CorporationID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if err := st.qRW.UpdateEveCharacter(ctx, arg); err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) UpdateEveCharacterName(ctx context.Context, characterID int32, name string) error {
	if characterID == 0 || name == "" {
		return fmt.Errorf("UpdateEveCharacterName: %w", app.ErrInvalid)
	}
	if err := st.qRW.UpdateEveCharacterName(ctx, queries.UpdateEveCharacterNameParams{
		ID:   int64(characterID),
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
		Description:    character.Description,
		Gender:         character.Gender,
		Faction:        eveEntityFromNullableDBModel(faction),
		ID:             int32(character.ID),
		Name:           character.Name,
		Race:           eveRaceFromDBModel(race),
		SecurityStatus: character.SecurityStatus,
		Title:          character.Title,
	}
	return &o
}
