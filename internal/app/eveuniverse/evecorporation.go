package eveuniverse

import (
	"context"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func (s *EveUniverseService) GetEveCorporation(ctx context.Context, corporationID int32) (*app.EveCorporation, error) {
	c, _, err := s.esiClient.ESI.CorporationApi.GetCorporationsCorporationId(ctx, corporationID, nil)
	if err != nil {
		return nil, err
	}
	ids := []int32{c.CreatorId, c.CeoId}
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
	o.Ceo, err = s.GetEveEntity(ctx, c.CeoId)
	if err != nil {
		return nil, err
	}
	o.Creator, err = s.GetEveEntity(ctx, c.CreatorId)
	if err != nil {
		return nil, err
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
