package storage

import (
	"context"
	"example/evebuddy/internal/storage/queries"
	"fmt"
)

func (r *Storage) CreateMailList(ctx context.Context, characterID, mailListID int32) error {
	arg := queries.CreateMailListParams{CharacterID: int64(characterID), EveEntityID: int64(mailListID)}
	if err := r.q.CreateMailList(ctx, arg); err != nil {
		return fmt.Errorf("failed to create mail list %d for character %d: %w", mailListID, characterID, err)
	}
	return nil
}

func (r *Storage) DeleteObsoleteMailLists(ctx context.Context, characterID int32) error {
	arg := queries.DeleteObsoleteMailListsParams{
		CharacterID:   int64(characterID),
		CharacterID_2: int64(characterID),
		CharacterID_3: int64(characterID),
	}
	if err := r.q.DeleteObsoleteMailLists(ctx, arg); err != nil {
		return fmt.Errorf("failed to delete obsolete mail lists for character %d: %w", characterID, err)
	}
	return nil
}
