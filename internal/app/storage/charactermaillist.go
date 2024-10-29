package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) CreateCharacterMailList(ctx context.Context, characterID, mailListID int32) error {
	arg := queries.CreateCharacterMailListParams{CharacterID: int64(characterID), EveEntityID: int64(mailListID)}
	if err := st.q.CreateCharacterMailList(ctx, arg); err != nil {
		return fmt.Errorf("create mail list %d for character %d: %w", mailListID, characterID, err)
	}
	return nil
}

func (st *Storage) DeleteObsoleteCharacterMailLists(ctx context.Context, characterID int32) error {
	arg := queries.DeleteObsoleteCharacterMailListsParams{
		CharacterID:   int64(characterID),
		CharacterID_2: int64(characterID),
		CharacterID_3: int64(characterID),
	}
	if err := st.q.DeleteObsoleteCharacterMailLists(ctx, arg); err != nil {
		return fmt.Errorf("delete obsolete mail lists for character %d: %w", characterID, err)
	}
	return nil
}
