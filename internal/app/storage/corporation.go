// Package sqlite contains the logic for storing application data into a local SQLite database.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mattn/go-sqlite3"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
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
		if errors.Is(err, sql.ErrNoRows) {
			err = app.ErrNotFound
		}
		return nil, fmt.Errorf("get corporation %d: %w", corporationID, err)
	}
	o := corporationFromDBModel(r)
	return o, nil
}

func corporationFromDBModel(r queries.GetCorporationRow) *app.Corporation {
	ec := eveCorporationFromDBModel(eveCorporationFromDBModelParams{
		corporation: r.EveCorporation,
		ceo: nullEveEntry{
			ID:       r.EveCorporation.CeoID,
			Name:     r.CeoName,
			Category: r.CeoCategory,
		},
		creator: nullEveEntry{
			ID:       r.EveCorporation.CreatorID,
			Name:     r.CreatorName,
			Category: r.CreatorCategory,
		},
		alliance: nullEveEntry{
			ID:       r.EveCorporation.AllianceID,
			Name:     r.AllianceName,
			Category: r.AllianceCategory,
		},
		faction: nullEveEntry{
			ID:       r.EveCorporation.FactionID,
			Name:     r.FactionName,
			Category: r.FactionCategory,
		},
		station: nullEveEntry{
			ID:       r.EveCorporation.HomeStationID,
			Name:     r.StationName,
			Category: r.StationCategory,
		},
	})
	o := &app.Corporation{
		ID:          int32(r.EveCorporation.ID),
		Corporation: ec,
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
