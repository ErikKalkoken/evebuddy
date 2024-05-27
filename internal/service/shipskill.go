package service

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

func (s *Service) ListCharacterShipsAbilities(characterID int32) ([]model.CharacterShipAbility, error) {
	ctx := context.Background()
	return s.r.ListCharacterShipsAbilities(ctx, characterID)
}

func (s *Service) UpdateShipSkills() error {
	ctx := context.Background()
	return s.r.UpdateShipSkills(ctx)

}
