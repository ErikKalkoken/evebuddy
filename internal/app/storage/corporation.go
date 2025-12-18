package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/mattn/go-sqlite3"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateCorporationParams struct {
	AssetValue        optional.Optional[float64]
	ID                int32
	IsTrainingWatched bool
	HomeID            optional.Optional[int64]
	LastCloneJumpAt   optional.Optional[time.Time]
	LastLoginAt       optional.Optional[time.Time]
	LocationID        optional.Optional[int64]
	ShipID            optional.Optional[int32]
	TotalSP           optional.Optional[int]
	UnallocatedSP     optional.Optional[int]
	WalletBalance     optional.Optional[float64]
}

func (st *Storage) CreateCorporation(ctx context.Context, corporationID int32) error {
	if err := st.qRW.CreateCorporation(ctx, int64(corporationID)); err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
				err = app.ErrAlreadyExists
			}
		}
		return fmt.Errorf("create corporation %d: %w", corporationID, err)
	}
	return nil
}

func (st *Storage) DeleteCorporation(ctx context.Context, corporationID int32) error {
	err := st.qRW.DeleteCorporation(ctx, int64(corporationID))
	if err != nil {
		return fmt.Errorf("delete corporation %d: %w", corporationID, err)
	}
	return nil
}

func (st *Storage) GetCorporation(ctx context.Context, corporationID int32) (*app.Corporation, error) {
	r, err := st.qRO.GetCorporation(ctx, int64(corporationID))
	if err != nil {
		return nil, fmt.Errorf("get corporation %d: %w", corporationID, convertGetError(err))
	}
	o := corporationFromDBModel(r)
	return o, nil
}

func (st *Storage) GetAnyCorporation(ctx context.Context) (*app.Corporation, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetAnyCorporation: %w", err)
	}
	ids, err := st.ListCorporationIDs(ctx)
	if err != nil {
		return nil, wrapErr(err)
	}
	if ids.Size() == 0 {
		return nil, wrapErr(app.ErrNotFound)
	}
	var id int32
	for v := range ids.All() {
		id = v
		break
	}
	o, err := st.GetCorporation(ctx, id)
	if err != nil {
		return nil, wrapErr(err)
	}
	return o, nil
}

func corporationFromDBModel(r queries.GetCorporationRow) *app.Corporation {
	ec := eveCorporationFromDBModel(eveCorporationFromDBModelParams{
		corporation: r.EveCorporation,
		ceo: nullEveEntry{
			id:       r.EveCorporation.CeoID,
			name:     r.CeoName,
			category: r.CeoCategory,
		},
		creator: nullEveEntry{
			id:       r.EveCorporation.CreatorID,
			name:     r.CreatorName,
			category: r.CreatorCategory,
		},
		alliance: nullEveEntry{
			id:       r.EveCorporation.AllianceID,
			name:     r.AllianceName,
			category: r.AllianceCategory,
		},
		faction: nullEveEntry{
			id:       r.EveCorporation.FactionID,
			name:     r.FactionName,
			category: r.FactionCategory,
		},
		station: nullEveEntry{
			id:       r.EveCorporation.HomeStationID,
			name:     r.StationName,
			category: r.StationCategory,
		},
	})
	o := &app.Corporation{
		ID:             int32(r.EveCorporation.ID),
		EveCorporation: ec,
	}
	return o
}

func (st *Storage) GetOrCreateCorporation(ctx context.Context, corporationID int32) (*app.Corporation, error) {
	ee, err := func() (*app.Corporation, error) {
		var r queries.GetCorporationRow
		tx, err := st.dbRW.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		qtx := st.qRW.WithTx(tx)
		r, err = qtx.GetCorporation(ctx, int64(corporationID))
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
			err = qtx.CreateCorporation(ctx, int64(corporationID))
			if err != nil {
				return nil, err
			}
			r, err = qtx.GetCorporation(ctx, int64(corporationID))
			if err != nil {
				return nil, err
			}
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return corporationFromDBModel(r), nil
	}()
	if err != nil {
		return nil, fmt.Errorf("GetOrCreateCorporation: %d: %w", corporationID, err)
	}
	return ee, nil
}

// ListCorporationIDs returns the IDs or all corporations.
func (st *Storage) ListCorporationIDs(ctx context.Context) (set.Set[int32], error) {
	ids, err := st.qRO.ListCorporationIDs(ctx)
	if err != nil {
		return set.Set[int32]{}, fmt.Errorf("list corporation IDs: %w", err)
	}
	ids2 := set.Of(convertNumericSlice[int32](ids)...)
	return ids2, nil
}

// ListOrphanedCorporationIDs returns ID of corporations without any members.
func (st *Storage) ListOrphanedCorporationIDs(ctx context.Context) (set.Set[int32], error) {
	ids, err := st.qRO.ListOrphanedCorporationIDs(ctx)
	if err != nil {
		return set.Set[int32]{}, fmt.Errorf("list orphaned corporation IDs: %w", err)
	}
	ids2 := set.Of(convertNumericSlice[int32](ids)...)
	return ids2, nil
}

// ListCorporationsShort returns all corporations ordered by name.
func (st *Storage) ListCorporationsShort(ctx context.Context) ([]*app.EntityShort[int32], error) {
	rows, err := st.qRO.ListCorporationsShort(ctx)
	if err != nil {
		return nil, fmt.Errorf("list corporations short: %w", err)

	}
	cc := make([]*app.EntityShort[int32], 0)
	for _, r := range rows {
		cc = append(cc, &app.EntityShort[int32]{ID: int32(r.ID), Name: r.Name})
	}
	return cc, nil
}

// ListPrivilegedCorporationsShort returns a list of corporations
// where a user has at least one character exist with a role from requiredRoles.
func (st *Storage) ListPrivilegedCorporationsShort(ctx context.Context, requiredRoles set.Set[app.Role]) ([]*app.EntityShort[int32], error) {
	rows, err := st.qRO.ListPrivilegedCorporationsShort(ctx, roles2names(requiredRoles).Slice())
	if err != nil {
		return nil, fmt.Errorf("ListPrivilegedCorporationsShort: %w", err)

	}
	cc := make([]*app.EntityShort[int32], 0)
	for _, r := range rows {
		cc = append(cc, &app.EntityShort[int32]{ID: int32(r.ID), Name: r.Name})
	}
	return cc, nil
}
