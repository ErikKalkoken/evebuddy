package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	islices "example/evebuddy/internal/helper/slices"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage/queries"
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

func (r *Storage) CreateEveCharacter(ctx context.Context, arg CreateEveCharacterParams) error {
	if arg.ID == 0 {
		return fmt.Errorf("invalid EveCharacter ID %d", arg.ID)
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
	err := r.q.CreateEveCharacter(ctx, arg2)
	if err != nil {
		return fmt.Errorf("failed to create EveCharacter %v, %w", arg2, err)
	}
	return nil
}

func (r *Storage) DeleteEveCharacter(ctx context.Context, characterID int32) error {
	err := r.q.DeleteEveCharacter(ctx, int64(characterID))
	if err != nil {
		return fmt.Errorf("failed to delete EveCharacter %d: %w", characterID, err)
	}
	return nil
}

func (r *Storage) GetEveCharacter(ctx context.Context, characterID int32) (*model.EveCharacter, error) {
	row, err := r.q.GetEveCharacter(ctx, int64(characterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get EveCharacter %d: %w", characterID, err)
	}
	c := eveCharacterFromDBModel(
		row.EveCharacter,
		row.EveEntity,
		row.EveRace,
		row.EveCharacterAlliance,
		row.EveCharacterFaction,
	)
	return c, nil
}

func (r *Storage) ListEveCharacterIDs(ctx context.Context) ([]int32, error) {
	ids, err := r.q.ListEveCharacterIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list EveCharacterIDs: %w", err)
	}
	ids2 := islices.ConvertNumeric[int64, int32](ids)
	return ids2, nil
}

func (r *Storage) UpdateEveCharacter(ctx context.Context, c *model.EveCharacter) error {
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
	if err := r.q.UpdateEveCharacter(ctx, arg); err != nil {
		return fmt.Errorf("failed to update or create EveCharacter %d: %w", c.ID, err)
	}
	return nil
}

func eveCharacterFromDBModel(
	character queries.EveCharacter,
	corporation queries.EveEntity,
	race queries.EveRace,
	alliance queries.EveCharacterAlliance,
	faction queries.EveCharacterFaction,
) *model.EveCharacter {
	x := model.EveCharacter{
		Alliance:       eveEntityFromEveCharacterAlliance(alliance),
		Birthday:       character.Birthday,
		Corporation:    eveEntityFromDBModel(corporation),
		Description:    character.Description,
		Gender:         character.Gender,
		Faction:        eveEntityFromEveCharacterFaction(faction),
		ID:             int32(character.ID),
		Name:           character.Name,
		Race:           eveRaceFromDBModel(race),
		SecurityStatus: character.SecurityStatus,
		Title:          character.Title,
	}
	return &x
}

func eveEntityFromEveCharacterAlliance(e queries.EveCharacterAlliance) *model.EveEntity {
	if !e.ID.Valid {
		return nil
	}
	category := eveEntityCategoryFromDBModel(e.Category.String)
	return &model.EveEntity{
		Category: category,
		ID:       int32(e.ID.Int64),
		Name:     e.Name.String,
	}
}

func eveEntityFromEveCharacterFaction(e queries.EveCharacterFaction) *model.EveEntity {
	if !e.ID.Valid {
		return nil
	}
	category := eveEntityCategoryFromDBModel(e.Category.String)
	return &model.EveEntity{
		Category: category,
		ID:       int32(e.ID.Int64),
		Name:     e.Name.String,
	}
}
