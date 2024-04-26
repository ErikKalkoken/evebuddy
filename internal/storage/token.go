package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage/queries"
)

func tokenFromDBModel(t queries.Token) model.Token {
	if t.MyCharacterID == 0 {
		panic("missing character ID")
	}
	return model.Token{
		AccessToken:  t.AccessToken,
		CharacterID:  int32(t.MyCharacterID),
		ExpiresAt:    t.ExpiresAt,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
	}
}

func (r *Storage) GetToken(ctx context.Context, characterID int32) (model.Token, error) {
	t, err := r.q.GetToken(ctx, int64(characterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return model.Token{}, fmt.Errorf("failed to get token for character %d: %w", characterID, err)
	}
	t2 := tokenFromDBModel(t)
	return t2, nil
}

func (r *Storage) UpdateOrCreateToken(ctx context.Context, t *model.Token) error {
	arg := queries.UpdateOrCreateTokenParams{
		AccessToken:   t.AccessToken,
		MyCharacterID: int64(t.CharacterID),
		ExpiresAt:     t.ExpiresAt,
		RefreshToken:  t.RefreshToken,
		TokenType:     t.TokenType,
	}
	if err := r.q.UpdateOrCreateToken(ctx, arg); err != nil {
		return fmt.Errorf("failed to update or create token for character %d: %w", t.CharacterID, err)
	}
	return nil
}
