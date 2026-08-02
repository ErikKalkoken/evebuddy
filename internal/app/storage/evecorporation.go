package storage

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func (st *Storage) GetEveCorporation(ctx context.Context, corporationID int64) (*app.EveCorporation, error) {
	r, err := st.qRO.GetEveCorporation(ctx, corporationID)
	if err != nil {
		return nil, fmt.Errorf("get EveCorporation %d: %w", corporationID, convertGetError(err))
	}
	c := eveCorporationFromDBModel(
		r.EveCorporation,
		nullCEO{
			id:       r.EveCorporation.CeoID,
			name:     r.CeoName,
			category: r.CeoCategory,
		},
		nullCreator{
			id:       r.EveCorporation.CreatorID,
			name:     r.CreatorName,
			category: r.CreatorCategory,
		},
		nullAlliance{
			id:       r.EveCorporation.AllianceID,
			name:     r.AllianceName,
			category: r.AllianceCategory,
		},
		nullFaction{
			id:       r.EveCorporation.FactionID,
			name:     r.FactionName,
			category: r.FactionCategory,
		},
		nullStation{
			id:       r.EveCorporation.HomeStationID,
			name:     r.StationName,
			category: r.StationCategory,
		},
	)
	return c, nil
}

func eveCorporationFromDBModel(
	corporation queries.EveCorporation,
	ceo nullCEO,
	creator nullCreator,
	alliance nullAlliance,
	faction nullFaction,
	station nullStation,
) *app.EveCorporation {
	o := &app.EveCorporation{
		ID:          corporation.ID,
		Alliance:    eveEntityFromNullableDBModel(nullEveEntity(alliance)),
		Ceo:         eveEntityFromNullableDBModel(nullEveEntity(ceo)),
		Creator:     eveEntityFromNullableDBModel(nullEveEntity(creator)),
		DateFounded: optional.FromNullTime(corporation.DateFounded),
		Description: corporation.Description,
		Faction:     eveEntityFromNullableDBModel(nullEveEntity(faction)),
		HomeStation: eveEntityFromNullableDBModel(nullEveEntity(station)),
		MemberCount: corporation.MemberCount,
		Name:        corporation.Name,
		Shares:      optional.FromNullInt64(corporation.Shares),
		TaxRate:     corporation.TaxRate,
		Ticker:      corporation.Ticker,
		URL:         optional.FromZeroValue(corporation.Url),
		WarEligible: optional.FromZeroValue(corporation.WarEligible),
	}
	return o
}

func (st *Storage) ListEveCorporationIDs(ctx context.Context) (set.Set[int64], error) {
	ids, err := st.qRO.ListEveCorporationIDs(ctx)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("ListEveCorporationIDs: %w", err)
	}
	ids2 := set.Collect(slices.Values(ids))
	return ids2, nil
}

type UpdateOrCreateEveCorporationParams struct {
	AllianceID    optional.Optional[int64]
	CeoID         optional.Optional[int64]
	CreatorID     optional.Optional[int64]
	DateFounded   optional.Optional[time.Time]
	Description   optional.Optional[string]
	FactionID     optional.Optional[int64]
	HomeStationID optional.Optional[int64]
	ID            int64
	MemberCount   int64
	Name          string
	Shares        optional.Optional[int64]
	TaxRate       float64
	Ticker        string
	URL           optional.Optional[string]
	WarEligible   optional.Optional[bool]
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
		Description:   arg.Description.ValueOrZero(),
		FactionID:     optional.ToNullInt64(arg.FactionID),
		HomeStationID: optional.ToNullInt64(arg.HomeStationID),
		ID:            arg.ID,
		MemberCount:   arg.MemberCount,
		Name:          arg.Name,
		Shares:        optional.ToNullInt64(arg.Shares),
		TaxRate:       float64(arg.TaxRate),
		Ticker:        arg.Ticker,
		Url:           arg.URL.ValueOrZero(),
		WarEligible:   arg.WarEligible.ValueOrZero(),
	}
	err := st.qRW.UpdateOrCreateEveCorporation(ctx, arg2)
	if err != nil {
		return fmt.Errorf("UpdateOrCreateEveCorporation: %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) UpdateEveCorporationName(ctx context.Context, corporationID int64, name string) error {
	if corporationID == 0 || name == "" {
		return fmt.Errorf("UpdateEveCorporationName: %w", app.ErrInvalid)
	}
	if err := st.qRW.UpdateEveCorporationName(ctx, queries.UpdateEveCorporationNameParams{
		ID:   corporationID,
		Name: name,
	}); err != nil {
		return fmt.Errorf("UpdateEveCorporationName %d: %w", corporationID, err)
	}
	return nil
}
