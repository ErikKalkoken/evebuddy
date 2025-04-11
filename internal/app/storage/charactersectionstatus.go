package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CharacterSectionStatusParams struct {
	CharacterID int32
	Section     app.CharacterSection

	CompletedAt time.Time
	ContentHash string
	Error       string
	StartedAt   time.Time
}

func (st *Storage) GetCharacterSectionStatus(ctx context.Context, characterID int32, section app.CharacterSection) (*app.CharacterSectionStatus, error) {
	arg := queries.GetCharacterSectionStatusParams{
		CharacterID: int64(characterID),
		SectionID:   string(section),
	}
	s, err := st.qRO.GetCharacterSectionStatus(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = app.ErrNotFound
		}
		return nil, fmt.Errorf("get status for character %d with section %s: %w", characterID, section, err)
	}
	s2 := characterSectionStatusFromDBModel(s)
	return s2, nil
}

func (st *Storage) ListCharacterSectionStatus(ctx context.Context, characterID int32) ([]*app.CharacterSectionStatus, error) {
	rows, err := st.qRO.ListCharacterSectionStatus(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list character status for ID %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterSectionStatus, len(rows))
	for i, row := range rows {
		oo[i] = characterSectionStatusFromDBModel(row)
	}
	return oo, nil
}

type UpdateOrCreateCharacterSectionStatusParams struct {
	// mandatory
	CharacterID int32
	Section     app.CharacterSection
	// optional
	CompletedAt  *sql.NullTime
	ContentHash  *string
	ErrorMessage *string
	StartedAt    *optional.Optional[time.Time]
}

func (st *Storage) UpdateOrCreateCharacterSectionStatus(ctx context.Context, arg UpdateOrCreateCharacterSectionStatusParams) (*app.CharacterSectionStatus, error) {
	if arg.CharacterID == 0 || arg.Section == "" {
		return nil, fmt.Errorf("UpdateOrCreateCharacterSectionStatus: %+v: %w", arg, app.ErrInvalid)
	}
	o, err := func() (*app.CharacterSectionStatus, error) {
		tx, err := st.dbRW.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		qtx := st.qRW.WithTx(tx)
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
		arg2.UpdatedAt = time.Now().UTC()
		if arg.CompletedAt != nil {
			arg2.CompletedAt = *arg.CompletedAt
		}
		if arg.ContentHash != nil {
			arg2.ContentHash = *arg.ContentHash
		}
		if arg.ErrorMessage != nil {
			arg2.Error = *arg.ErrorMessage
		}
		if arg.StartedAt != nil {
			arg2.StartedAt = optional.ToNullTime(*arg.StartedAt)
		}
		o, err := qtx.UpdateOrCreateCharacterSectionStatus(ctx, arg2)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return characterSectionStatusFromDBModel(o), nil
	}()
	if err != nil {
		return nil, fmt.Errorf("update or create status for character %d and section %s: %w", arg.CharacterID, arg.Section, err)
	}
	return o, nil
}

func characterSectionStatusFromDBModel(o queries.CharacterSectionStatus) *app.CharacterSectionStatus {
	x := &app.CharacterSectionStatus{
		ID:           o.ID,
		CharacterID:  int32(o.CharacterID),
		ErrorMessage: o.Error,
		Section:      app.CharacterSection(o.SectionID),
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
