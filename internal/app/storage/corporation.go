package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/mattn/go-sqlite3"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CreateCorporationParams struct {
	AssetValue        optional.Optional[float64]
	ID                int64
	IsTrainingWatched bool
	HomeID            optional.Optional[int64]
	LastCloneJumpAt   optional.Optional[time.Time]
	LastLoginAt       optional.Optional[time.Time]
	LocationID        optional.Optional[int64]
	ShipID            optional.Optional[int64]
	TotalSP           optional.Optional[int]
	UnallocatedSP     optional.Optional[int]
	WalletBalance     optional.Optional[float64]
}

func (st *Storage) CreateCorporation(ctx context.Context, corporationID int64) error {
	if err := st.qRW.CreateCorporation(ctx, corporationID); err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
				err = app.ErrAlreadyExists
			}
		}
		return fmt.Errorf("create corporation %d: %w", corporationID, err)
	}
	return nil
}

func (st *Storage) DeleteCorporation(ctx context.Context, corporationID int64) error {
	err := st.qRW.DeleteCorporation(ctx, corporationID)
	if err != nil {
		return fmt.Errorf("delete corporation %d: %w", corporationID, err)
	}
	return nil
}

func (st *Storage) GetCorporation(ctx context.Context, corporationID int64) (*app.Corporation, error) {
	r, err := st.qRO.GetCorporation(ctx, corporationID)
	if err != nil {
		return nil, fmt.Errorf("get corporation %d: %w", corporationID, convertGetError(err))
	}
	o := corporationFromGetCorporationRow(r)
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
	var id int64
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

// corporationFromDBModel takes fields individually, not a row struct,
// so it works across queries whose sqlc-generated row types differ
// despite embedding EveCorporation the same way.
func corporationFromDBModel(
	ec queries.EveCorporation,
	ceoName, ceoCategory sql.NullString,
	creatorName, creatorCategory sql.NullString,
	allianceName, allianceCategory sql.NullString,
	factionName, factionCategory sql.NullString,
	stationName, stationCategory sql.NullString,
) *app.Corporation {
	o := eveCorporationFromDBModel(
		ec,
		nullCEO{
			id:       ec.CeoID,
			name:     ceoName,
			category: ceoCategory,
		},
		nullCreator{
			id:       ec.CreatorID,
			name:     creatorName,
			category: creatorCategory,
		},
		nullAlliance{
			id:       ec.AllianceID,
			name:     allianceName,
			category: allianceCategory,
		},
		nullFaction{
			id:       ec.FactionID,
			name:     factionName,
			category: factionCategory,
		},
		nullStation{
			id:       ec.HomeStationID,
			name:     stationName,
			category: stationCategory,
		},
	)
	return &app.Corporation{
		ID:             ec.ID,
		EveCorporation: o,
	}
}

func corporationFromGetCorporationRow(r queries.GetCorporationRow) *app.Corporation {
	return corporationFromDBModel(
		r.EveCorporation,
		r.CeoName, r.CeoCategory,
		r.CreatorName, r.CreatorCategory,
		r.AllianceName, r.AllianceCategory,
		r.FactionName, r.FactionCategory,
		r.StationName, r.StationCategory,
	)
}

func corporationFromListCorporationsRow(r queries.ListCorporationsRow) *app.Corporation {
	return corporationFromDBModel(
		r.EveCorporation,
		r.CeoName, r.CeoCategory,
		r.CreatorName, r.CreatorCategory,
		r.AllianceName, r.AllianceCategory,
		r.FactionName, r.FactionCategory,
		r.StationName, r.StationCategory,
	)
}

func (st *Storage) GetOrCreateCorporation(ctx context.Context, corporationID int64) (*app.Corporation, error) {
	ee, err := func() (*app.Corporation, error) {
		var r queries.GetCorporationRow
		tx, err := st.dbRW.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		qtx := st.qRW.WithTx(tx)
		r, err = qtx.GetCorporation(ctx, corporationID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
			err = qtx.CreateCorporation(ctx, corporationID)
			if err != nil {
				return nil, err
			}
			r, err = qtx.GetCorporation(ctx, corporationID)
			if err != nil {
				return nil, err
			}
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return corporationFromGetCorporationRow(r), nil
	}()
	if err != nil {
		return nil, fmt.Errorf("GetOrCreateCorporation: %d: %w", corporationID, err)
	}
	return ee, nil
}

// ListCorporationIDs returns the IDs or all corporations.
func (st *Storage) ListCorporationIDs(ctx context.Context) (set.Set[int64], error) {
	ids, err := st.qRO.ListCorporationIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list corporation IDs: %w", err)
	}
	ids2 := set.Collect(slices.Values(ids))
	return ids2, nil
}

// ListOrphanedCorporationIDs returns ID of corporations without any members.
func (st *Storage) ListOrphanedCorporationIDs(ctx context.Context) (set.Set[int64], error) {
	ids, err := st.qRO.ListOrphanedCorporationIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list orphaned corporation IDs: %w", err)
	}
	ids2 := set.Collect(slices.Values(ids))
	return ids2, nil
}

// ListCorporations returns all corporations.
func (st *Storage) ListCorporations(ctx context.Context) ([]*app.Corporation, error) {
	rows, err := st.qRO.ListCorporations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list corporations: %w", err)
	}
	oo := make([]*app.Corporation, len(rows))
	for i, r := range rows {
		oo[i] = corporationFromListCorporationsRow(r)
	}
	return oo, nil
}

func (st *Storage) ListCorporationsShort(ctx context.Context) ([]*app.EntityShort, error) {
	rows, err := st.qRO.ListCorporationsShort(ctx)
	if err != nil {
		return nil, fmt.Errorf("list corporations short: %w", err)

	}
	var cc []*app.EntityShort
	for _, r := range rows {
		cc = append(cc, &app.EntityShort{ID: r.ID, Name: r.Name})
	}
	return cc, nil
}

// ListPrivilegedCorporationsShort returns a list of corporations
// where a user has at least one character exist with a role from requiredRoles.
func (st *Storage) ListPrivilegedCorporationsShort(ctx context.Context, requiredRoles set.Set[app.Role]) ([]*app.EntityShort, error) {
	rows, err := st.qRO.ListPrivilegedCorporationsShort(ctx, slices.Collect(roles2names(requiredRoles).All()))
	if err != nil {
		return nil, fmt.Errorf("ListPrivilegedCorporationsShort: %w", err)

	}
	var cc []*app.EntityShort
	for _, r := range rows {
		cc = append(cc, &app.EntityShort{ID: r.ID, Name: r.Name})
	}
	return cc, nil
}
