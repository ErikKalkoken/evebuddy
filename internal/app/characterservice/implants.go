package characterservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func (s *CharacterService) ListImplants(ctx context.Context, characterID int32) ([]*app.CharacterImplant, error) {
	return s.st.ListCharacterImplants(ctx, characterID)
}

func (s *CharacterService) updateImplantsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionCharacterImplants {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			implants, _, err := s.esiClient.ESI.ClonesApi.GetCharactersCharacterIdImplants(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			slog.Debug("Received implants from ESI", "count", len(implants), "characterID", characterID)
			return implants, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			implants := data.([]int32)
			if err := s.eus.AddMissingTypes(ctx, set.Of(implants...)); err != nil {
				return err
			}
			args := make([]storage.CreateCharacterImplantParams, len(implants))
			for i, typeID := range implants {
				args[i] = storage.CreateCharacterImplantParams{
					CharacterID: characterID,
					EveTypeID:   typeID,
				}
			}
			if err := s.st.ReplaceCharacterImplants(ctx, characterID, args); err != nil {
				return err
			}
			slog.Info("Stored updated implants", "characterID", characterID, "count", len(implants))
			return nil
		})
}
