package eveuniverse

import (
	"context"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func (s *EveUniverseService) GetEveCorporationESI(ctx context.Context, corporationID int32) (*app.EveCorporation, error) {
	x, _, err := s.esiClient.ESI.CorporationApi.GetCorporationsCorporationId(ctx, corporationID, nil)
	if err != nil {
		return nil, err
	}
	_, err = s.AddMissingEveEntities(ctx, []int32{corporationID, x.CeoId, x.CreatorId, x.AllianceId, x.FactionId, x.HomeStationId})
	if err != nil {
		return nil, err
	}
	o := &app.EveCorporation{
		DateFounded: x.DateFounded,
		Description: x.Description,
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
	if x.CeoId != 0 && x.CeoId != 1 {
		o.Ceo, err = s.GetEveEntity(ctx, x.CeoId)
		if err != nil {
			return nil, err
		}
	}
	o.Creator, err = s.getValidEveEntity(ctx, x.CreatorId)
	if err != nil {
		return nil, err
	}
	o.Alliance, err = s.getValidEveEntity(ctx, x.AllianceId)
	if err != nil {
		return nil, err
	}
	o.Faction, err = s.getValidEveEntity(ctx, x.FactionId)
	if err != nil {
		return nil, err
	}
	o.HomeStation, err = s.getValidEveEntity(ctx, x.HomeStationId)
	if err != nil {
		return nil, err
	}
	return o, nil
}
