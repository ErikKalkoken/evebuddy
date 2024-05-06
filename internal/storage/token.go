package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

func tokenFromDBModel(t queries.Token, scopes []string) *model.Token {
	if t.MyCharacterID == 0 {
		panic("missing character ID")
	}
	return &model.Token{
		AccessToken:  t.AccessToken,
		CharacterID:  int32(t.MyCharacterID),
		ExpiresAt:    t.ExpiresAt,
		RefreshToken: t.RefreshToken,
		Scopes:       scopes,
		TokenType:    t.TokenType,
	}
}

func (r *Storage) GetToken(ctx context.Context, characterID int32) (*model.Token, error) {
	t, err := r.q.GetToken(ctx, int64(characterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get token for character %d: %w", characterID, err)
	}
	ss, err := r.q.ListTokenScopes(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	scopes := make([]string, len(ss))
	for i, s := range ss {
		scopes[i] = s.Name
	}
	t2 := tokenFromDBModel(t, scopes)
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
	ss := make([]queries.Scope, len(t.Scopes))
	for i, name := range t.Scopes {
		s, err := r.getOrCreateScope(ctx, name)
		if err != nil {
			return err
		}
		ss[i] = s
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := r.q.WithTx(tx)
	if err := qtx.ClearTokenScopes(ctx, int64(t.CharacterID)); err != nil {
		return err
	}
	for _, s := range ss {
		arg := queries.AddTokenScopeParams{
			TokenID: arg.MyCharacterID,
			ScopeID: s.ID,
		}
		if err := qtx.AddTokenScope(ctx, arg); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *Storage) getOrCreateScope(ctx context.Context, name string) (queries.Scope, error) {
	var s queries.Scope
	if name == "" {
		return s, fmt.Errorf("invalid scope name")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return s, err
	}
	defer tx.Rollback()
	qtx := r.q.WithTx(tx)
	s, err = qtx.GetScope(ctx, name)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return s, err
		}
		s, err = qtx.CreateScope(ctx, name)
		if err != nil {
			return s, err
		}
	}
	if err := tx.Commit(); err != nil {
		return s, err
	}
	return s, nil
}
