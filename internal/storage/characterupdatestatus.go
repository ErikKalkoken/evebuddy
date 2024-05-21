package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CharacterUpdateStatusParams struct {
	CharacterID   int32
	Section       model.CharacterSection
	Error         string
	LastUpdatedAt sql.NullTime
	ContentHash   string
}

func (r *Storage) GetCharacterUpdateStatus(ctx context.Context, characterID int32, section model.CharacterSection) (*model.CharacterUpdateStatus, error) {
	arg := queries.GetCharacterUpdateStatusParams{
		CharacterID: int64(characterID),
		SectionID:   string(section),
	}
	s, err := r.q.GetCharacterUpdateStatus(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get update status for character %d with section %s: %w", characterID, section, err)
	}
	s2 := characterUpdateStatusFromDBModel(s)
	return s2, nil
}

func (r *Storage) ListCharacterUpdateStatus(ctx context.Context, characterID int32) ([]*model.CharacterUpdateStatus, error) {
	rows, err := r.q.ListCharacterUpdateStatus(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("failed to list character update status for ID %c: %w", characterID, err)
	}
	oo := make([]*model.CharacterUpdateStatus, len(rows))
	for i, row := range rows {
		oo[i] = characterUpdateStatusFromDBModel(row)
	}
	return oo, nil
}

func (r *Storage) SetCharacterUpdateStatusError(ctx context.Context, characterID int32, section model.CharacterSection, errorText string) error {
	arg := queries.SetCharacterUpdateStatusParams{
		CharacterID: int64(characterID),
		SectionID:   string(section),
		Error:       errorText,
	}
	return r.q.SetCharacterUpdateStatus(ctx, arg)
}

func (r *Storage) UpdateOrCreateCharacterUpdateStatus(ctx context.Context, arg CharacterUpdateStatusParams) error {
	arg1 := queries.UpdateOrCreateCharacterUpdateStatusParams{
		CharacterID:   int64(arg.CharacterID),
		SectionID:     string(arg.Section),
		Error:         arg.Error,
		LastUpdatedAt: arg.LastUpdatedAt,
		ContentHash:   arg.ContentHash,
	}
	err := r.q.UpdateOrCreateCharacterUpdateStatus(ctx, arg1)
	if err != nil {
		return fmt.Errorf("failed to update or create updates status for character %d with section %s: %w", arg.CharacterID, arg.Section, err)
	}
	return nil
}

func characterUpdateStatusFromDBModel(o queries.CharacterUpdateStatus) *model.CharacterUpdateStatus {
	return &model.CharacterUpdateStatus{
		ID:            o.ID,
		CharacterID:   int32(o.CharacterID),
		ErrorMessage:  o.Error,
		Section:       model.CharacterSection(o.SectionID),
		LastUpdatedAt: o.LastUpdatedAt,
		ContentHash:   o.ContentHash,
	}
}
