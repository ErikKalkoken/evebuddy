package characters

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

func (s *Characters) ListCharacterShipsAbilities(ctx context.Context, characterID int32, search string) ([]*model.CharacterShipAbility, error) {
	return s.st.ListCharacterShipsAbilities(ctx, characterID, search)
}
