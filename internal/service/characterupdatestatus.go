package service

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

func (s *Service) ListCharacterUpdateStatus(characterID int32) ([]*model.CharacterUpdateStatus, error) {
	ctx := context.Background()
	return s.r.ListCharacterUpdateStatus(ctx, characterID)
}
