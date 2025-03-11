package eveuniverse

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// CharacterCorporationHistory returns a list of all the corporations a character has been a member of in descending order.
func (s *EveUniverseService) CharacterCorporationHistory(ctx context.Context, characterID int32) ([]app.MembershipHistoryItem, error) {
	items, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterIdCorporationhistory(ctx, characterID, nil)
	if err != nil {
		return nil, err
	}
	items2 := make([]organizationHistoryItem, len(items))
	for i, it := range items {
		items2[i] = organizationHistoryItem{
			OrganizationID: it.CorporationId,
			IsDeleted:      it.IsDeleted,
			RecordID:       int(it.RecordId),
			StartDate:      it.StartDate,
		}
	}
	return s.makeMembershipHistory(ctx, items2)
}

// CharacterCorporationHistory returns a list of all the alliances a corporation has been a member of in descending order.
func (s *EveUniverseService) CorporationAllianceHistory(ctx context.Context, corporationID int32) ([]app.MembershipHistoryItem, error) {
	items, _, err := s.esiClient.ESI.CorporationApi.GetCorporationsCorporationIdAlliancehistory(ctx, corporationID, nil)
	if err != nil {
		return nil, err
	}
	items2 := make([]organizationHistoryItem, len(items))
	for i, it := range items {
		items2[i] = organizationHistoryItem{
			OrganizationID: it.AllianceId,
			IsDeleted:      it.IsDeleted,
			RecordID:       int(it.RecordId),
			StartDate:      it.StartDate,
		}
	}
	return s.makeMembershipHistory(ctx, items2)
}

type organizationHistoryItem struct {
	OrganizationID int32
	IsDeleted      bool
	RecordID       int
	StartDate      time.Time
}

func (s *EveUniverseService) makeMembershipHistory(ctx context.Context, items []organizationHistoryItem) ([]app.MembershipHistoryItem, error) {
	ids := make([]int32, 0)
	for _, it := range items {
		ids = append(ids, it.OrganizationID)
	}
	_, err := s.AddMissingEveEntities(ctx, ids)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b organizationHistoryItem) int {
		return cmp.Compare(a.RecordID, b.RecordID)
	})

	oo := make([]app.MembershipHistoryItem, len(items))
	for i, it := range items {
		corporation, err := s.GetEveEntity(ctx, it.OrganizationID)
		if err != nil {
			return nil, err
		}
		var endDate time.Time
		if i+1 < len(items) {
			endDate = items[i+1].StartDate
		}
		var endDate2 time.Time
		if !endDate.IsZero() {
			endDate2 = endDate
		} else {
			endDate2 = s.Now()
		}
		days := int(endDate2.Sub(it.StartDate) / (time.Hour * 24))
		oo[i] = app.MembershipHistoryItem{
			Days:         days,
			EndDate:      endDate,
			IsDeleted:    it.IsDeleted,
			Organization: corporation,
			RecordID:     it.RecordID,
			StartDate:    it.StartDate,
		}
	}
	slices.SortFunc(oo, func(a, b app.MembershipHistoryItem) int {
		return -cmp.Compare(a.RecordID, b.RecordID)
	})
	return oo, nil
}
