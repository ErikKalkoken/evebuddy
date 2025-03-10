package character

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/antihax/goesi/esi"
)

// CorporationHistory returns a list of all the corporations a character has been a member of in descending order.
func (s *CharacterService) CorporationHistory(ctx context.Context, characterID int32) ([]app.CharacterCorporationHistoryItem, error) {
	items, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterIdCorporationhistory(ctx, characterID, nil)
	if err != nil {
		return nil, err
	}
	ids := make([]int32, 0)
	for _, it := range items {
		ids = append(ids, it.CorporationId)
	}
	_, err = s.EveUniverseService.AddMissingEveEntities(ctx, ids)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b esi.GetCharactersCharacterIdCorporationhistory200Ok) int {
		return cmp.Compare(a.RecordId, b.RecordId)
	})

	oo := make([]app.CharacterCorporationHistoryItem, len(items))
	for i, it := range items {
		corporation, err := s.EveUniverseService.GetEveEntity(ctx, it.CorporationId)
		if err != nil {
			return nil, err
		}
		var endDate time.Time
		if i+1 < len(items) {
			endDate = items[i+1].StartDate
		}
		oo[i] = app.CharacterCorporationHistoryItem{
			CharaterID:  characterID,
			Corporation: corporation,
			IsDeleted:   it.IsDeleted,
			RecordID:    int(it.RecordId),
			StartDate:   it.StartDate,
			EndDate:     endDate,
		}
	}
	slices.SortFunc(oo, func(a, b app.CharacterCorporationHistoryItem) int {
		return -cmp.Compare(a.RecordID, b.RecordID)
	})
	return oo, nil
}
