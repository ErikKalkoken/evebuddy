package storage

import (
	"context"
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
		return nil, fmt.Errorf("get EveCorporation %d: %w", corporationID, convertGetError(err))
	}
	c := eveCorporationFromDBModel(eveCorporationFromDBModelParams{
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
	if arg.ID == 0 {
		return fmt.Errorf("update or create corporation industry job: %+v: invalid parameters", arg)
	}
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

func (st *Storage) UpdateEveCorporationName(ctx context.Context, corporationID int32, name string) error {
	if corporationID == 0 || name == "" {
		return fmt.Errorf("UpdateEveCorporationName: %w", app.ErrInvalid)
	}
	if err := st.qRW.UpdateEveCorporationName(ctx, queries.UpdateEveCorporationNameParams{
		ID:   int64(corporationID),
		Name: name,
	}); err != nil {
		return fmt.Errorf("UpdateEveCorporationName %d: %w", corporationID, err)
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
