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

func (st *Storage) GetGeneralSectionStatus(ctx context.Context, section app.GeneralSection) (*app.GeneralSectionStatus, error) {
	s, err := st.qRO.GetGeneralSectionStatus(ctx, string(section))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = app.ErrNotFound
		}
		return nil, fmt.Errorf("get status for general section %s: %w", section, err)
	}
	s2 := generalSectionStatusFromDBModel(s)
	return s2, nil
}

func (st *Storage) ListGeneralSectionStatus(ctx context.Context) ([]*app.GeneralSectionStatus, error) {
	rows, err := st.qRO.ListGeneralSectionStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("list general section status: %w", err)
	}
	oo := make([]*app.GeneralSectionStatus, len(rows))
	for i, row := range rows {
		oo[i] = generalSectionStatusFromDBModel(row)
	}
	return oo, nil
}

type UpdateOrCreateGeneralSectionStatusParams struct {
	// Mandatory
	Section app.GeneralSection
	// Optional
	CompletedAt *sql.NullTime
	ContentHash *string
	Error       *string
	StartedAt   *optional.Optional[time.Time]
}

func (st *Storage) UpdateOrCreateGeneralSectionStatus(ctx context.Context, arg UpdateOrCreateGeneralSectionStatusParams) (*app.GeneralSectionStatus, error) {
	if string(arg.Section) == "" {
		return nil, fmt.Errorf("UpdateOrCreateGeneralSectionStatus: %+v: %w", arg, app.ErrInvalid)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
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
		arg2.StartedAt = optional.ToNullTime(*arg.StartedAt)
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

func generalSectionStatusFromDBModel(o queries.GeneralSectionStatus) *app.GeneralSectionStatus {
	x := &app.GeneralSectionStatus{
		ErrorMessage: o.Error,
		Section:      app.GeneralSection(o.SectionID),
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
