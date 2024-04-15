package repository

import (
	"context"
	"example/evebuddy/internal/sqlc"
)

func (r *Repository) CreateMailList(ctx context.Context, characterID, mailListID int32) error {
	arg2 := sqlc.CreateMailListParams{CharacterID: int64(characterID), EveEntityID: int64(mailListID)}
	if err := r.q.CreateMailList(ctx, arg2); err != nil {
		return err
	}
	return nil
}
