package eveuniverse

import (
	"context"
	"slices"
	"strings"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func (s *EveUniverseService) GetAllianceESI(ctx context.Context, allianceID int32) (*app.EveAlliance, error) {
	a, _, err := s.esiClient.ESI.AllianceApi.GetAlliancesAllianceId(ctx, allianceID, nil)
	if err != nil {
		return nil, err
	}
	_, err = s.AddMissingEntities(ctx, []int32{allianceID, a.CreatorCorporationId, a.CreatorId, a.ExecutorCorporationId, a.FactionId})
	if err != nil {
		return nil, err
	}
	o := &app.EveAlliance{
		DateFounded: a.DateFounded,
		ID:          allianceID,
		Name:        a.Name,
		Ticker:      a.Ticker,
	}
	o.CreatorCorporation, err = s.getValidEveEntity(ctx, a.CreatorCorporationId)
	if err != nil {
		return nil, err
	}
	o.Creator, err = s.getValidEveEntity(ctx, a.CreatorId)
	if err != nil {
		return nil, err
	}
	o.ExecutorCorporation, err = s.getValidEveEntity(ctx, a.ExecutorCorporationId)
	if err != nil {
		return nil, err
	}
	o.Faction, err = s.getValidEveEntity(ctx, a.FactionId)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *EveUniverseService) GetAllianceCorporationsESI(ctx context.Context, allianceID int32) ([]*app.EveEntity, error) {
	ids, _, err := s.esiClient.ESI.AllianceApi.GetAlliancesAllianceIdCorporations(ctx, allianceID, nil)
	if err != nil {
		return nil, err
	}
	_, err = s.AddMissingEntities(ctx, slices.Concat(ids, []int32{allianceID}))
	if err != nil {
		return nil, err
	}
	oo, err := s.st.ListEveEntitiesForIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(oo, func(a, b *app.EveEntity) int {
		return strings.Compare(a.Name, b.Name)
	})
	return oo, nil
}
