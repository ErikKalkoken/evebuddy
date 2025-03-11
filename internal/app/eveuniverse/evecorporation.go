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
	ids := make([]int32, 0)
	if c.CeoId != 0 && c.CeoId != 1 {
		ids = append(ids, c.CeoId)
	}
	if c.CreatorId != 0 && c.CreatorId != 1 {
		ids = append(ids, c.CreatorId)
	}
	if c.AllianceId != 0 {
		ids = append(ids, c.AllianceId)
	}
	if c.FactionId != 0 {
		ids = append(ids, c.FactionId)
	}
	if c.HomeStationId != 0 {
		ids = append(ids, c.HomeStationId)
	}
	_, err = s.AddMissingEveEntities(ctx, ids)
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
	if c.CreatorId != 0 && c.CreatorId != 1 {
		o.Creator, err = s.GetEveEntity(ctx, c.CreatorId)
		if err != nil {
			return nil, err
		}
	}
	if c.AllianceId != 0 {
		o.Alliance, err = s.GetEveEntity(ctx, c.AllianceId)
		if err != nil {
			return nil, err
		}
	}
	if c.FactionId != 0 {
		o.Faction, err = s.GetEveEntity(ctx, c.FactionId)
		if err != nil {
			return nil, err
		}
	}
	if c.HomeStationId != 0 {
		o.HomeStation, err = s.GetEveEntity(ctx, c.HomeStationId)
		if err != nil {
			return nil, err
		}
	}
	return o, nil
}
