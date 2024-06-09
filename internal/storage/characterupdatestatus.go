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

type CharacterUpdateStatusParams struct {
	CharacterID int32
	Section     model.CharacterSection

	CompletedAt time.Time
	ContentHash string
	Error       string
	StartedAt   time.Time
}

func (st *Storage) GetCharacterUpdateStatus(ctx context.Context, characterID int32, section model.CharacterSection) (*model.CharacterUpdateStatus, error) {
	arg := queries.GetCharacterUpdateStatusParams{
		CharacterID: int64(characterID),
		SectionID:   string(section),
	}
	s, err := st.q.GetCharacterUpdateStatus(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get update status for character %d with section %s: %w", characterID, section, err)
	}
	s2 := characterUpdateStatusFromDBModel(s)
	return s2, nil
}

func (st *Storage) ListCharacterUpdateStatus(ctx context.Context, characterID int32) ([]*model.CharacterUpdateStatus, error) {
	rows, err := st.q.ListCharacterUpdateStatus(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("failed to list character update status for ID %d: %w", characterID, err)
	}
	oo := make([]*model.CharacterUpdateStatus, len(rows))
	for i, row := range rows {
		oo[i] = characterUpdateStatusFromDBModel(row)
	}
	return oo, nil
}

type CharacterUpdateStatusOptionals struct {
	CompletedAt sql.NullTime
	ContentHash sql.NullString
	Error       sql.NullString
	StartedAt   sql.NullTime
}

func (st *Storage) UpdateOrCreateCharacterUpdateStatus2(ctx context.Context, characterID int32, section model.CharacterSection, arg CharacterUpdateStatusOptionals) (*model.CharacterUpdateStatus, error) {
	if characterID == 0 || section == "" {
		panic("Invalid params")
	}
	tx, err := st.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	qtx := st.q.WithTx(tx)
	var arg2 queries.UpdateOrCreateCharacterUpdateStatusParams
	old, err := qtx.GetCharacterUpdateStatus(ctx, queries.GetCharacterUpdateStatusParams{
		CharacterID: int64(characterID),
		SectionID:   string(section),
	})
	if errors.Is(err, sql.ErrNoRows) {
		arg2 = queries.UpdateOrCreateCharacterUpdateStatusParams{
			CharacterID: int64(characterID),
			SectionID:   string(section),
		}
	} else if err != nil {
		return nil, err
	} else {
		arg2 = queries.UpdateOrCreateCharacterUpdateStatusParams{
			CharacterID: int64(characterID),
			SectionID:   string(section),
			CompletedAt: old.CompletedAt,
			ContentHash: old.ContentHash,
			Error:       old.Error,
			StartedAt:   old.StartedAt,
		}
	}
	if arg.CompletedAt.Valid {
		arg2.CompletedAt = arg.CompletedAt
	}
	if arg.ContentHash.Valid {
		arg2.ContentHash = arg.ContentHash.String
	}
	if arg.Error.Valid {
		arg2.Error = arg.Error.String
	}
	if arg.StartedAt.Valid {
		arg2.StartedAt = arg.StartedAt
	}
	o, err := qtx.UpdateOrCreateCharacterUpdateStatus(ctx, arg2)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return characterUpdateStatusFromDBModel(o), nil
}

func (st *Storage) UpdateOrCreateCharacterUpdateStatus(ctx context.Context, arg CharacterUpdateStatusParams) (*model.CharacterUpdateStatus, error) {
	arg2 := characterUpdateStatusDBModelFromParams(arg)
	o, err := st.q.UpdateOrCreateCharacterUpdateStatus(ctx, arg2)
	if err != nil {
		return nil, fmt.Errorf("failed to update or create updates status for character %d with section %s: %w", arg.CharacterID, arg.Section, err)
	}
	return characterUpdateStatusFromDBModel(o), nil
}

func characterUpdateStatusDBModelFromParams(arg CharacterUpdateStatusParams) queries.UpdateOrCreateCharacterUpdateStatusParams {
	arg2 := queries.UpdateOrCreateCharacterUpdateStatusParams{
		CharacterID: int64(arg.CharacterID),
		ContentHash: arg.ContentHash,
		Error:       arg.Error,
		SectionID:   string(arg.Section),
	}
	if !arg.CompletedAt.IsZero() {
		arg2.CompletedAt = sql.NullTime{Time: arg.CompletedAt, Valid: true}
	}
	if !arg.StartedAt.IsZero() {
		arg2.StartedAt = sql.NullTime{Time: arg.StartedAt, Valid: true}
	}
	return arg2
}

func characterUpdateStatusFromDBModel(o queries.CharacterUpdateStatus) *model.CharacterUpdateStatus {
	x := &model.CharacterUpdateStatus{
		ID:           o.ID,
		CharacterID:  int32(o.CharacterID),
		ErrorMessage: o.Error,
		Section:      model.CharacterSection(o.SectionID),
		ContentHash:  o.ContentHash,
	}
	if o.CompletedAt.Valid {
		x.CompletedAt = o.CompletedAt.Time
	}
	if o.StartedAt.Valid {
		x.StartedAt = o.StartedAt.Time
	}
	return x
}
