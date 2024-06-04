package character

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

func (s *CharacterService) ListCharacterShipsAbilities(ctx context.Context, characterID int32, search string) ([]*model.CharacterShipAbility, error) {
	return s.st.ListCharacterShipsAbilities(ctx, characterID, search)
}

func (s *CharacterService) ListCharacterShipSkills(ctx context.Context, characterID, shipTypeID int32) ([]*model.CharacterShipSkill, error) {
	return s.st.ListCharacterShipSkills(ctx, characterID, shipTypeID)
}
