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

type CorporationSectionParams struct {
	CorporationID int32
	Section       app.CorporationSection
}

func (x CorporationSectionParams) IsInvalid() bool {
	return x.CorporationID == 0
}

func (st *Storage) ResetCorporationSectionStatusContentHash(ctx context.Context, arg CorporationSectionParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("ResetCorporationSectionStatusContentHash: %+v: %w", arg, err)
	}
	if arg.IsInvalid() {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateCorporationSectionStatusContentHash(ctx, queries.UpdateCorporationSectionStatusContentHashParams{
		ContentHash:   "",
		CorporationID: int64(arg.CorporationID),
		SectionID:     arg.Section.String(),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetCorporationSectionStatus(ctx context.Context, corporationID int32, section app.CorporationSection) (*app.CorporationSectionStatus, error) {
	arg := queries.GetCorporationSectionStatusParams{
		CorporationID: int64(corporationID),
		SectionID:     section.String(),
	}
	s, err := st.qRO.GetCorporationSectionStatus(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf(
			"get status for corporation %d with section %s: %w",
			corporationID,
			section,
			convertGetError(err),
		)
	}
	s2 := corporationSectionStatusFromDBModel(s)
	return s2, nil
}

func (st *Storage) ListCorporationSectionStatus(ctx context.Context, corporationID int32) ([]*app.CorporationSectionStatus, error) {
	rows, err := st.qRO.ListCorporationSectionStatus(ctx, int64(corporationID))
	if err != nil {
		return nil, fmt.Errorf("list corporation status for ID %d: %w", corporationID, err)
	}
	oo := make([]*app.CorporationSectionStatus, len(rows))
	for i, row := range rows {
		oo[i] = corporationSectionStatusFromDBModel(row)
	}
	return oo, nil
}

type UpdateOrCreateCorporationSectionStatusParams struct {
	// mandatory
	CorporationID int32
	Section       app.CorporationSection
	// optional
	Comment      *string
	CompletedAt  *sql.NullTime
	ContentHash  *string
	ErrorMessage *string
	StartedAt    *optional.Optional[time.Time]
}

func (st *Storage) UpdateOrCreateCorporationSectionStatus(ctx context.Context, arg UpdateOrCreateCorporationSectionStatusParams) (*app.CorporationSectionStatus, error) {
	if arg.CorporationID == 0 || arg.Section == "" {
		return nil, fmt.Errorf("UpdateOrCreateCorporationSectionStatus: %+v: %w", arg, app.ErrInvalid)
	}
	o, err := func() (*app.CorporationSectionStatus, error) {
		tx, err := st.dbRW.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		qtx := st.qRW.WithTx(tx)
		var arg2 queries.UpdateOrCreateCorporationSectionStatusParams
		old, err := qtx.GetCorporationSectionStatus(ctx, queries.GetCorporationSectionStatusParams{
			CorporationID: int64(arg.CorporationID),
			SectionID:     arg.Section.String(),
		})
		if errors.Is(err, sql.ErrNoRows) {
			arg2 = queries.UpdateOrCreateCorporationSectionStatusParams{
				CorporationID: int64(arg.CorporationID),
				SectionID:     arg.Section.String(),
			}
		} else if err != nil {
			return nil, err
		} else {
			arg2 = queries.UpdateOrCreateCorporationSectionStatusParams{
				CorporationID: int64(arg.CorporationID),
				SectionID:     arg.Section.String(),
				CompletedAt:   old.CompletedAt,
				ContentHash:   old.ContentHash,
				Error:         old.Error,
				StartedAt:     old.StartedAt,
			}
		}
		arg2.UpdatedAt = time.Now().UTC()
		if arg.Comment != nil {
			arg2.Comment = *arg.Comment
		}
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
		o, err := qtx.UpdateOrCreateCorporationSectionStatus(ctx, arg2)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return corporationSectionStatusFromDBModel(o), nil
	}()
	if err != nil {
		return nil, fmt.Errorf("update or create status for corporation %d and section %s: %w", arg.CorporationID, arg.Section, err)
	}
	return o, nil
}

func corporationSectionStatusFromDBModel(o queries.CorporationSectionStatus) *app.CorporationSectionStatus {
	x := &app.CorporationSectionStatus{
		Comment:       o.Comment,
		CorporationID: int32(o.CorporationID),
		SectionStatus: app.SectionStatus{
			ContentHash:  o.ContentHash,
			ErrorMessage: o.Error,
			Section:      app.CorporationSection(o.SectionID),
			UpdatedAt:    o.UpdatedAt,
		},
	}
	if o.CompletedAt.Valid {
		x.CompletedAt = o.CompletedAt.Time
	}
	if o.StartedAt.Valid {
		x.StartedAt = o.StartedAt.Time
	}
	return x
}
