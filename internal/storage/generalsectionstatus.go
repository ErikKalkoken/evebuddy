package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

func (st *Storage) GetGeneralSectionStatus(ctx context.Context, section model.GeneralSection) (*model.GeneralSectionStatus, error) {
	s, err := st.q.GetGeneralSectionStatus(ctx, string(section))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get status for general section %s: %w", section, err)
	}
	s2 := generalSectionStatusFromDBModel(s)
	return s2, nil
}

func (st *Storage) ListGeneralSectionStatus(ctx context.Context) ([]*model.GeneralSectionStatus, error) {
	rows, err := st.q.ListGeneralSectionStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list general section status: %w", err)
	}
	oo := make([]*model.GeneralSectionStatus, len(rows))
	for i, row := range rows {
		oo[i] = generalSectionStatusFromDBModel(row)
	}
	return oo, nil
}

type UpdateOrCreateGeneralSectionStatusParams struct {
	// Mandatory
	Section model.GeneralSection
	// Optional
	CompletedAt *sql.NullTime
	ContentHash *string
	Error       *string
	StartedAt   *sql.NullTime
}

func (st *Storage) UpdateOrCreateGeneralSectionStatus(ctx context.Context, arg UpdateOrCreateGeneralSectionStatusParams) (*model.GeneralSectionStatus, error) {
	if string(arg.Section) == "" {
		panic("Invalid params")
	}
	tx, err := st.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	qtx := st.q.WithTx(tx)
	var arg2 queries.UpdateOrCreateGeneralSectionStatusParams
	old, err := qtx.GetGeneralSectionStatus(ctx, string(arg.Section))
	if errors.Is(err, sql.ErrNoRows) {
		arg2 = queries.UpdateOrCreateGeneralSectionStatusParams{
			SectionID: string(arg.Section),
		}
	} else if err != nil {
		return nil, err
	} else {
		arg2 = queries.UpdateOrCreateGeneralSectionStatusParams{
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
	o, err := qtx.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return generalSectionStatusFromDBModel(o), nil
}

func generalSectionStatusFromDBModel(o queries.GeneralSectionStatus) *model.GeneralSectionStatus {
	x := &model.GeneralSectionStatus{
		ID:           o.ID,
		ErrorMessage: o.Error,
		Section:      model.GeneralSection(o.SectionID),
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
