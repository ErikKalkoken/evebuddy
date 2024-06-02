package characters

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

func (s *Characters) ListCharacterShipsAbilities(ctx context.Context, characterID int32, search string) ([]*model.CharacterShipAbility, error) {
	return s.r.ListCharacterShipsAbilities(ctx, characterID, search)
}

func (s *Characters) UpdateShipSkills(ctx context.Context) error {
	return s.r.UpdateShipSkills(ctx)

}
