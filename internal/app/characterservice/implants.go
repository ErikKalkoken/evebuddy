package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func (s *CharacterService) ListImplants(ctx context.Context, characterID int64) ([]*app.CharacterImplant, error) {
	return s.st.ListCharacterImplants(ctx, characterID)
}

func (s *CharacterService) ListAllImplants(ctx context.Context) ([]*app.CharacterImplant, error) {
	return s.st.ListAllCharacterImplants(ctx)
}

func (s *CharacterService) updateImplantsESI(ctx context.Context, arg characterSectionUpdateParams) (bool, error) {
	if arg.section != app.SectionCharacterImplants {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg, true,
		func(ctx context.Context, characterID int64) (any, error) {
			ctx = xgoesi.NewContextWithOperationID(ctx, "GetCharactersCharacterIdImplants")
			implants, _, err := s.esiClient.ClonesAPI.GetCharactersCharacterIdImplants(ctx, characterID).Execute()
			if err != nil {
				return false, err
			}
			slog.Debug("Received implants from ESI", "count", len(implants), "characterID", characterID)
			return implants, nil
		},
		func(ctx context.Context, characterID int64, data any) (bool, error) {
			implants := data.([]int64)
			incoming := set.Collect(slices.Values(implants))
			current, err := s.st.ListCharacterImplantIDs(ctx, characterID)
			if err != nil {
				return false, err
			}
			if current.Equal(incoming) {
				return false, nil
			}

			if err := s.eus.AddMissingTypes(ctx, incoming); err != nil {
				return false, err
			}
			if err := s.st.ReplaceCharacterImplants(ctx, characterID, incoming); err != nil {
				return false, err
			}
			slog.Info("Stored updated implants", "characterID", characterID, "count", incoming.Size())
			return true, nil
		})
}
