package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func (s *CharacterService) ListAllLoyaltyPointEntries(ctx context.Context) ([]*app.CharacterLoyaltyPointEntry, error) {
	return s.st.ListAllCharacterLoyaltyPointEntries(ctx)
}

func (s *CharacterService) ListLoyaltyPointEntries(ctx context.Context, characterID int64) ([]*app.CharacterLoyaltyPointEntry, error) {
	return s.st.ListCharacterLoyaltyPointEntries(ctx, characterID)
}

func (s *CharacterService) updateLoyaltyPointEntriesESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterLoyaltyPoints {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdLoyaltyPoints")
			rows, _, err := s.esiClient.LoyaltyAPI.GetCharactersCharacterIdLoyaltyPoints(ctx, characterID).Execute()
			if err != nil {
				return nil, err
			}
			slog.Debug("Received loyalty points entries from ESI", "count", len(rows), "characterID", characterID)
			return rows, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			incoming := make(map[int64]int64)
			for _, r := range data.([]esi.CharactersCharacterIdLoyaltyPointsGetInner) {
				incoming[r.CorporationId] = r.LoyaltyPoints
			}

			entries, err := s.st.ListCharacterLoyaltyPointEntries(ctx, characterID)
			if err != nil {
				return false, err
			}
			current := make(map[int64]int64)
			for _, o := range entries {
				current[o.Corporation.ID] = o.LoyaltyPoints
			}
			if maps.Equal(incoming, current) {
				return false, nil
			}

			var added, updated set.Set[int64]
			for id, points := range incoming {
				points2, ok := current[id]
				if !ok {
					added.Add(id)
					continue
				}
				if points != points2 {
					updated.Add(id)
					continue
				}
			}

			for id := range added.All() {
				_, err := s.eus.GetOrCreateCorporationESI(ctx, id)
				if err != nil {
					slog.Error("Failed to get corporation for loyalty point entry", "corporationID", id, "error", err)
					continue
				}
				err = s.st.UpdateOrCreateCharacterLoyaltyPointEntry(ctx, storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
					CharacterID:   characterID,
					CorporationID: id,
					LoyaltyPoints: incoming[id],
				})
				if err != nil {
					return false, err
				}
			}
			slog.Info("Added loyalty points entries", "characterID", characterID, "count", added.Size())

			for id := range updated.All() {
				err = s.st.UpdateOrCreateCharacterLoyaltyPointEntry(ctx, storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
					CharacterID:   characterID,
					CorporationID: id,
					LoyaltyPoints: incoming[id],
				})
				if err != nil {
					return false, err
				}
			}
			slog.Info("Updated loyalty points entries", "characterID", characterID, "count", updated.Size())

			// Delete obsolete entries
			currentIDs, err := s.st.ListCharacterLoyaltyPointEntryIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			incomingIDs := set.Collect(maps.Keys(incoming))
			obsoleteIDs := set.Difference(currentIDs, incomingIDs)
			if obsoleteIDs.Size() > 0 {
				err := s.st.DeleteCharacterLoyaltyPointEntries(ctx, characterID, obsoleteIDs)
				if err != nil {
					return false, err
				}
				slog.Info("Deleted obsolete loyalty points entries", "characterID", characterID, "count", obsoleteIDs.Size())
			}
			return true, nil
		})
}
