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
	CompletedAt time.Time
	ContentHash string
	Error       string
	Section     model.CharacterSection
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
		return nil, fmt.Errorf("failed to list character update status for ID %c: %w", characterID, err)
	}
	oo := make([]*model.CharacterUpdateStatus, len(rows))
	for i, row := range rows {
		oo[i] = characterUpdateStatusFromDBModel(row)
	}
	return oo, nil
}

func (st *Storage) SetCharacterUpdateStatusError(ctx context.Context, characterID int32, section model.CharacterSection, errorText string) error {
	arg := queries.SetCharacterUpdateStatusParams{
		CharacterID: int64(characterID),
		SectionID:   string(section),
		Error:       errorText,
	}
	return st.q.SetCharacterUpdateStatus(ctx, arg)
}

func (st *Storage) UpdateOrCreateCharacterUpdateStatus(ctx context.Context, arg CharacterUpdateStatusParams) error {
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
	err := st.q.UpdateOrCreateCharacterUpdateStatus(ctx, arg2)
	if err != nil {
		return fmt.Errorf("failed to update or create updates status for character %d with section %s: %w", arg.CharacterID, arg.Section, err)
	}
	return nil
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
