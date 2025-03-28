package eveuniverseservice

import (
	"context"
	"slices"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func (s *EveUniverseService) GetCorporationESI(ctx context.Context, corporationID int32) (*app.EveCorporation, error) {
	x, _, err := s.esiClient.ESI.CorporationApi.GetCorporationsCorporationId(ctx, corporationID, nil)
	if err != nil {
		return nil, err
	}
	ids := slices.DeleteFunc(
		[]int32{corporationID, x.CeoId, x.CreatorId, x.AllianceId, x.FactionId, x.HomeStationId},
		func(id int32) bool {
			return id < 2
		})
	eeMap, err := s.ToEntities(ctx, ids)
	if err != nil {
		return nil, err
	}
	o := &app.EveCorporation{
		Alliance:    eeMap[x.AllianceId],
		Ceo:         eeMap[x.CeoId],
		Creator:     eeMap[x.CreatorId],
		Faction:     eeMap[x.FactionId],
		DateFounded: x.DateFounded,
		Description: x.Description,
		HomeStation: eeMap[x.HomeStationId],
		ID:          corporationID,
		MemberCount: int(x.MemberCount),
		Name:        x.Name,
		Shares:      int(x.Shares),
		TaxRate:     x.TaxRate,
		Ticker:      x.Ticker,
		URL:         x.Url,
		WarEligible: x.WarEligible,
		Timestamp:   time.Now().UTC(),
	}
	return o, nil
}
