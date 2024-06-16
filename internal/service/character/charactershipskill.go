package character

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func (s *CharacterService) ListCharacterShipsAbilities(ctx context.Context, characterID int32, search string) ([]*app.CharacterShipAbility, error) {
	return s.st.ListCharacterShipsAbilities(ctx, characterID, search)
}

func (s *CharacterService) ListCharacterShipSkills(ctx context.Context, characterID, shipTypeID int32) ([]*app.CharacterShipSkill, error) {
	return s.st.ListCharacterShipSkills(ctx, characterID, shipTypeID)
}
