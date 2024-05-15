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

type MyCharacterUpdateStatusParams struct {
	MyCharacterID int32
	Section       model.UpdateSection
	UpdatedAt     time.Time
	ContentHash   string
}

func (r *Storage) GetMyCharacterUpdateStatus(ctx context.Context, characterID int32, section model.UpdateSection) (*model.MyCharacterUpdateStatus, error) {
	arg := queries.GetMyCharacterUpdateStatusParams{
		MyCharacterID: int64(characterID),
		SectionID:     string(section),
	}
	s, err := r.q.GetMyCharacterUpdateStatus(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get update status for character %d with section %s: %w", characterID, section, err)
	}
	s2 := myCharacterUpdateStatusFromDBModel(s)
	return s2, nil
}

func (r *Storage) UpdateOrCreateMyCharacterUpdateStatus(ctx context.Context, arg MyCharacterUpdateStatusParams) error {
	arg1 := queries.UpdateOrCreateMyCharacterUpdateStatusParams{
		MyCharacterID: int64(arg.MyCharacterID),
		SectionID:     string(arg.Section),
		UpdatedAt:     arg.UpdatedAt,
		ContentHash:   arg.ContentHash,
	}
	err := r.q.UpdateOrCreateMyCharacterUpdateStatus(ctx, arg1)
	if err != nil {
		return fmt.Errorf("failed to update or create updates status for character %d with section %s: %w", arg.MyCharacterID, arg.Section, err)
	}
	return nil
}

func myCharacterUpdateStatusFromDBModel(s queries.MyCharacterUpdateStatus) *model.MyCharacterUpdateStatus {
	return &model.MyCharacterUpdateStatus{
		MyCharacterID: int32(s.MyCharacterID),
		SectionID:     model.UpdateSection(s.SectionID),
		UpdatedAt:     s.UpdatedAt,
		ContentHash:   s.ContentHash,
	}
}
