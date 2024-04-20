package storage

import (
	"context"
	"example/evebuddy/internal/storage/sqlc"
	"fmt"
)

func (r *Storage) CreateMailList(ctx context.Context, characterID, mailListID int32) error {
	arg2 := sqlc.CreateMailListParams{CharacterID: int64(characterID), EveEntityID: int64(mailListID)}
	if err := r.q.CreateMailList(ctx, arg2); err != nil {
		return fmt.Errorf("failed to create mail list %d for character %d: %w", mailListID, characterID, err)
	}
	return nil
}
