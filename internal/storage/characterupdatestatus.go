package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CharacterSectionStatusParams struct {
	CharacterID int32
	Section     model.CharacterSection

	CompletedAt time.Time
	ContentHash string
	Error       string
	StartedAt   time.Time
}

func (st *Storage) GetCharacterSectionStatus(ctx context.Context, characterID int32, section model.CharacterSection) (*model.CharacterSectionStatus, error) {
	arg := queries.GetCharacterSectionStatusParams{
		CharacterID: int64(characterID),
		SectionID:   string(section),
	}
	s, err := st.q.GetCharacterSectionStatus(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get update status for character %d with section %s: %w", characterID, section, err)
	}
	s2 := characterSectionStatusFromDBModel(s)
	return s2, nil
}

func (st *Storage) ListCharacterSectionStatus(ctx context.Context, characterID int32) ([]*model.CharacterSectionStatus, error) {
	rows, err := st.q.ListCharacterSectionStatus(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("failed to list character update status for ID %d: %w", characterID, err)
	}
	oo := make([]*model.CharacterSectionStatus, len(rows))
	for i, row := range rows {
		oo[i] = characterSectionStatusFromDBModel(row)
	}
	return oo, nil
}

type CharacterSectionStatusOptionals struct {
	CompletedAt *sql.NullTime
	ContentHash *string
	Error       *string
	StartedAt   *sql.NullTime
}

func (st *Storage) UpdateOrCreateCharacterSectionStatus2(ctx context.Context, characterID int32, section model.CharacterSection, optionals CharacterSectionStatusOptionals) (*model.CharacterSectionStatus, error) {
	if characterID == 0 || section == "" {
		panic("Invalid params")
	}
	tx, err := st.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	qtx := st.q.WithTx(tx)
	var arg2 queries.UpdateOrCreateCharacterSectionStatusParams
	old, err := qtx.GetCharacterSectionStatus(ctx, queries.GetCharacterSectionStatusParams{
		CharacterID: int64(characterID),
		SectionID:   string(section),
	})
	if errors.Is(err, sql.ErrNoRows) {
		arg2 = queries.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID: int64(characterID),
			SectionID:   string(section),
		}
	} else if err != nil {
		return nil, err
	} else {
		arg2 = queries.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID: int64(characterID),
			SectionID:   string(section),
			CompletedAt: old.CompletedAt,
			ContentHash: old.ContentHash,
			Error:       old.Error,
			StartedAt:   old.StartedAt,
		}
	}
	if optionals.CompletedAt != nil {
		arg2.CompletedAt = *optionals.CompletedAt
	}
	if optionals.ContentHash != nil {
		arg2.ContentHash = *optionals.ContentHash
	}
	if optionals.Error != nil {
		arg2.Error = *optionals.Error
	}
	if optionals.StartedAt != nil {
		arg2.StartedAt = *optionals.StartedAt
	}
	o, err := qtx.UpdateOrCreateCharacterSectionStatus(ctx, arg2)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return characterSectionStatusFromDBModel(o), nil
}

func (st *Storage) UpdateOrCreateCharacterSectionStatus(ctx context.Context, arg CharacterSectionStatusParams) (*model.CharacterSectionStatus, error) {
	arg2 := characterSectionStatusDBModelFromParams(arg)
	o, err := st.q.UpdateOrCreateCharacterSectionStatus(ctx, arg2)
	if err != nil {
		return nil, fmt.Errorf("failed to update or create updates status for character %d with section %s: %w", arg.CharacterID, arg.Section, err)
	}
	return characterSectionStatusFromDBModel(o), nil
}

func characterSectionStatusDBModelFromParams(arg CharacterSectionStatusParams) queries.UpdateOrCreateCharacterSectionStatusParams {
	arg2 := queries.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID: int64(arg.CharacterID),
		ContentHash: arg.ContentHash,
		Error:       arg.Error,
		SectionID:   string(arg.Section),
		UpdatedAt:   time.Now(),
	}
	if !arg.CompletedAt.IsZero() {
		arg2.CompletedAt = sql.NullTime{Time: arg.CompletedAt, Valid: true}
	}
	if !arg.StartedAt.IsZero() {
		arg2.StartedAt = sql.NullTime{Time: arg.StartedAt, Valid: true}
	}
	return arg2
}

func characterSectionStatusFromDBModel(o queries.CharacterSectionStatus) *model.CharacterSectionStatus {
	x := &model.CharacterSectionStatus{
		ID:           o.ID,
		CharacterID:  int32(o.CharacterID),
		ErrorMessage: o.Error,
		Section:      model.CharacterSection(o.SectionID),
		ContentHash:  o.ContentHash,
		UpdatedAt:    o.UpdatedAt,
	}
	if o.CompletedAt.Valid {
		x.CompletedAt = o.CompletedAt.Time
	}
	if o.StartedAt.Valid {
		x.StartedAt = o.StartedAt.Time
	}
	return x
}
