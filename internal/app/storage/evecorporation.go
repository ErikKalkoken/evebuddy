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
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func (st *Storage) GetEveCorporation(ctx context.Context, corporationID int32) (*app.EveCorporation, error) {
	r, err := st.qRO.GetEveCorporation(ctx, int64(corporationID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = app.ErrNotFound
		}
		return nil, fmt.Errorf("get EveCorporation %d: %w", corporationID, err)
	}
	c := eveCorporationFromDBModel(eveCorporationFromDBModelParams{
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
	return c, nil
}

func (st *Storage) ListEveCorporationIDs(ctx context.Context) (set.Set[int32], error) {
	ids, err := st.qRO.ListEveCorporationIDs(ctx)
	if err != nil {
		return set.Set[int32]{}, fmt.Errorf("ListEveCorporationIDs: %w", err)
	}
	ids2 := set.Of(convertNumericSlice[int32](ids)...)
	return ids2, nil
}

type UpdateOrCreateEveCorporationParams struct {
	AllianceID    optional.Optional[int32]
	CeoID         optional.Optional[int32]
	CreatorID     optional.Optional[int32]
	DateFounded   optional.Optional[time.Time]
	Description   string
	FactionID     optional.Optional[int32]
	HomeStationID optional.Optional[int32]
	ID            int32
	MemberCount   int32
	Name          string
	Shares        optional.Optional[int64]
	TaxRate       float32
	Ticker        string
	URL           string
	WarEligible   bool
}

func (st *Storage) UpdateOrCreateEveCorporation(ctx context.Context, arg UpdateOrCreateEveCorporationParams) error {
	arg2 := queries.UpdateOrCreateEveCorporationParams{
		AllianceID:    optional.ToNullInt64(arg.AllianceID),
		CeoID:         optional.ToNullInt64(arg.CeoID),
		CreatorID:     optional.ToNullInt64(arg.CreatorID),
		DateFounded:   optional.ToNullTime(arg.DateFounded),
		Description:   arg.Description,
		FactionID:     optional.ToNullInt64(arg.FactionID),
		HomeStationID: optional.ToNullInt64(arg.HomeStationID),
		ID:            int64(arg.ID),
		MemberCount:   int64(arg.MemberCount),
		Name:          arg.Name,
		Shares:        optional.ToNullInt64(arg.Shares),
		TaxRate:       float64(arg.TaxRate),
		Ticker:        arg.Ticker,
		Url:           arg.URL,
		WarEligible:   arg.WarEligible,
	}
	err := st.qRW.UpdateOrCreateEveCorporation(ctx, arg2)
	if err != nil {
		return fmt.Errorf("UpdateOrCreateEveCorporation: %+v: %w", arg, err)
	}
	return nil
}

type eveCorporationFromDBModelParams struct {
	corporation queries.EveCorporation
	ceo         nullEveEntry
	creator     nullEveEntry
	alliance    nullEveEntry
	faction     nullEveEntry
	station     nullEveEntry
}

func eveCorporationFromDBModel(arg eveCorporationFromDBModelParams) *app.EveCorporation {
	o := &app.EveCorporation{
		ID:          int32(arg.corporation.ID),
		Alliance:    eveEntityFromNullableDBModel(arg.alliance),
		Ceo:         eveEntityFromNullableDBModel(arg.ceo),
		Creator:     eveEntityFromNullableDBModel(arg.creator),
		DateFounded: optional.FromNullTime(arg.corporation.DateFounded),
		Description: arg.corporation.Description,
		Faction:     eveEntityFromNullableDBModel(arg.faction),
		HomeStation: eveEntityFromNullableDBModel(arg.station),
		MemberCount: int(arg.corporation.MemberCount),
		Name:        arg.corporation.Name,
		Shares:      optional.FromNullInt64ToInteger[int](arg.corporation.Shares),
		TaxRate:     float32(arg.corporation.TaxRate),
		Ticker:      arg.corporation.Ticker,
		URL:         arg.corporation.Url,
		WarEligible: arg.corporation.WarEligible,
	}
	return o
}
