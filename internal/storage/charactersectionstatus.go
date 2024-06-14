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

type UpdateOrCreateCharacterSectionStatusParams struct {
	// mandatory
	CharacterID int32
	Section     model.CharacterSection
	// optional
	CompletedAt *sql.NullTime
	ContentHash *string
	Error       *string
	StartedAt   *sql.NullTime
}

func (st *Storage) UpdateOrCreateCharacterSectionStatus(ctx context.Context, arg UpdateOrCreateCharacterSectionStatusParams) (*model.CharacterSectionStatus, error) {
	if arg.CharacterID == 0 || arg.Section == "" {
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
		CharacterID: int64(arg.CharacterID),
		SectionID:   string(arg.Section),
	})
	if errors.Is(err, sql.ErrNoRows) {
		arg2 = queries.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID: int64(arg.CharacterID),
			SectionID:   string(arg.Section),
		}
	} else if err != nil {
		return nil, err
	} else {
		arg2 = queries.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID: int64(arg.CharacterID),
			SectionID:   string(arg.Section),
			CompletedAt: old.CompletedAt,
			ContentHash: old.ContentHash,
			Error:       old.Error,
			StartedAt:   old.StartedAt,
		}
	}
	if arg.CompletedAt != nil {
		arg2.CompletedAt = *arg.CompletedAt
	}
	if arg.ContentHash != nil {
		arg2.ContentHash = *arg.ContentHash
	}
	if arg.Error != nil {
		arg2.Error = *arg.Error
	}
	if arg.StartedAt != nil {
		arg2.StartedAt = *arg.StartedAt
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
