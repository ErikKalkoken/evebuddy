package characterservice

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func (s *CharacterService) ListCharacterShipsAbilities(ctx context.Context, characterID int32, search string) ([]*app.CharacterShipAbility, error) {
	return s.st.ListCharacterShipsAbilities(ctx, characterID, search)
}
