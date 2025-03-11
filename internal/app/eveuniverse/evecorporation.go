package eveuniverse

import (
	"context"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func (s *EveUniverseService) GetEveCorporationESI(ctx context.Context, corporationID int32) (*app.EveCorporation, error) {
	c, _, err := s.esiClient.ESI.CorporationApi.GetCorporationsCorporationId(ctx, corporationID, nil)
	if err != nil {
		return nil, err
	}
	_, err = s.AddMissingEveEntities(ctx, []int32{corporationID, c.CeoId, c.CreatorId, c.AllianceId, c.FactionId, c.HomeStationId})
	if err != nil {
		return nil, err
	}
	o := &app.EveCorporation{
		DateFounded: c.DateFounded,
		Description: c.Description,
		ID:          corporationID,
		MemberCount: int(c.MemberCount),
		Name:        c.Name,
		Shares:      int(c.Shares),
		TaxRate:     c.TaxRate,
		Ticker:      c.Ticker,
		URL:         c.Url,
		WarEligible: c.WarEligible,
		Timestamp:   time.Now().UTC(),
	}
	if c.CeoId != 0 && c.CeoId != 1 {
		o.Ceo, err = s.GetEveEntity(ctx, c.CeoId)
		if err != nil {
			return nil, err
		}
	}
	o.Creator, err = s.getValidEveEntity(ctx, c.CreatorId)
	if err != nil {
		return nil, err
	}
	o.Alliance, err = s.getValidEveEntity(ctx, c.AllianceId)
	if err != nil {
		return nil, err
	}
	o.Faction, err = s.getValidEveEntity(ctx, c.FactionId)
	if err != nil {
		return nil, err
	}
	o.HomeStation, err = s.getValidEveEntity(ctx, c.HomeStationId)
	if err != nil {
		return nil, err
	}
	return o, nil
}
