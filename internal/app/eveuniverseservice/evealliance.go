package eveuniverseservice

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func (s *EveUniverseService) GetAllianceESI(ctx context.Context, allianceID int32) (*app.EveAlliance, error) {
	a, _, err := s.esiClient.ESI.AllianceApi.GetAlliancesAllianceId(ctx, allianceID, nil)
	if err != nil {
		return nil, err
	}
	ids := slices.DeleteFunc(
		[]int32{allianceID, a.CreatorCorporationId, a.CreatorId, a.ExecutorCorporationId, a.FactionId},
		func(id int32) bool {
			return id < 2
		})
	eeMap, err := s.ToEveEntities(ctx, ids)
	if err != nil {
		return nil, err
	}
	maps.DeleteFunc(eeMap, func(id int32, o *app.EveEntity) bool {
		return !o.Category.IsKnown()
	})
	o := &app.EveAlliance{
		Creator:             eeMap[a.CreatorId],
		CreatorCorporation:  eeMap[a.CreatorCorporationId],
		DateFounded:         a.DateFounded,
		ExecutorCorporation: eeMap[a.ExecutorCorporationId],
		Faction:             eeMap[a.FactionId],
		ID:                  allianceID,
		Name:                a.Name,
		Ticker:              a.Ticker,
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
