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
	Section     model.UpdateSection
	UpdatedAt   time.Time
	ContentHash string
}

func (r *Storage) GetCharacterUpdateStatus(ctx context.Context, characterID int32, section model.UpdateSection) (*model.CharacterUpdateStatus, error) {
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

func (r *Storage) UpdateOrCreateCharacterUpdateStatus(ctx context.Context, arg CharacterUpdateStatusParams) error {
	arg1 := queries.UpdateOrCreateCharacterUpdateStatusParams{
		CharacterID: int64(arg.CharacterID),
		SectionID:   string(arg.Section),
		UpdatedAt:   arg.UpdatedAt,
		ContentHash: arg.ContentHash,
	}
	err := r.q.UpdateOrCreateCharacterUpdateStatus(ctx, arg1)
	if err != nil {
		return fmt.Errorf("failed to update or create updates status for character %d with section %s: %w", arg.CharacterID, arg.Section, err)
	}
	return nil
}

func characterUpdateStatusFromDBModel(s queries.CharacterUpdateStatus) *model.CharacterUpdateStatus {
	return &model.CharacterUpdateStatus{
		CharacterID: int32(s.CharacterID),
		SectionID:   model.UpdateSection(s.SectionID),
		UpdatedAt:   s.UpdatedAt,
		ContentHash: s.ContentHash,
	}
}
