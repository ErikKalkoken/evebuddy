package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func (s *CharacterService) ListAllLoyaltyPointEntries(ctx context.Context) ([]*app.CharacterLoyaltyPointEntry, error) {
	return s.st.ListAllCharacterLoyaltyPointEntries(ctx)
}

func (s *CharacterService) ListLoyaltyPointEntries(ctx context.Context, characterID int64) ([]*app.CharacterLoyaltyPointEntry, error) {
	return s.st.ListCharacterLoyaltyPointEntries(ctx, characterID)
}

func (s *CharacterService) updateLoyaltyPointEntriesESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterLoyaltyPoints {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
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
			points := make(map[int64]int64)
			for _, r := range data.([]esi.CharactersCharacterIdLoyaltyPointsGetInner) {
				points[r.CorporationId] = r.LoyaltyPoints
			}
			missing, err := s.st.MissingEveCorporations(ctx, set.Collect(maps.Keys(points)))
			if err != nil {
				return false, err
			}
			for id := range missing.All() {
				_, err := s.eus.GetOrCreateCorporationESI(ctx, id)
				if err != nil {
					if ctx.Err() != nil {
						return false, err
					}
					slog.Error("Failed to get corporation for loyalty point entry", "corporationID", id, "error", err)
					delete(points, id)
				}
			}
			return s.st.ReplaceCharacterLoyaltyPointEntries(ctx, characterID, points)
		})
}
