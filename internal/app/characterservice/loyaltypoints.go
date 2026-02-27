package characterservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
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
		ctx, arg,
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
			rows := data.([]esi.CharactersCharacterIdLoyaltyPointsGetInner)
			incoming := set.Collect(xiter.MapSlice(rows, func(x esi.CharactersCharacterIdLoyaltyPointsGetInner) int64 {
				return x.CorporationId
			}))
			for id := range incoming.All() {
				_, err := s.eus.GetOrCreateCorporationESI(ctx, id)
				if err != nil {
					return false, err
				}
			}
			for _, r := range rows {
				err := s.st.UpdateOrCreateCharacterLoyaltyPointEntry(ctx, storage.UpdateOrCreateCharacterLoyaltyPointEntryParams{
					CharacterID:   characterID,
					CorporationID: r.CorporationId,
					LoyaltyPoints: r.LoyaltyPoints,
				})
				if err != nil {
					return false, err
				}
			}
			slog.Info("Stored updated loyalty points", "characterID", characterID, "count", len(rows))

			// Delete obsolete entries
			if arg.MarketOrderRetention == 0 {
				return true, nil
			}

			current, err := s.st.ListCharacterLoyaltyPointEntryIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			obsolete := set.Difference(incoming, current)
			if obsolete.Size() > 0 {
				err := s.st.DeleteCharacterLoyaltyPointEntries(ctx, characterID, obsolete)
				if err != nil {
					return false, err
				}
				slog.Info("Deleted obsolete loyalty points entries", "characterID", characterID, "count", obsolete.Size())
			}
			return true, nil
		})
}
