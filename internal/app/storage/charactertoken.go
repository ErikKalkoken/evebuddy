package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) GetCharacterToken(ctx context.Context, characterID int32) (*app.CharacterToken, error) {
	t, err := st.qRO.GetCharacterToken(ctx, int64(characterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = app.ErrNotFound
		}
		return nil, fmt.Errorf("get token for character %d: %w", characterID, err)
	}
	ss, err := st.qRO.ListCharacterTokenScopes(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	scopes := make([]string, len(ss))
	for i, s := range ss {
		scopes[i] = s.Name
	}
	t2 := characterTokenFromDBModel(t, scopes)
	return t2, nil
}

func (st *Storage) UpdateOrCreateCharacterToken(ctx context.Context, t *app.CharacterToken) error {
	err := func() error {
		arg := queries.UpdateOrCreateCharacterTokenParams{
			AccessToken:  t.AccessToken,
			CharacterID:  int64(t.CharacterID),
			ExpiresAt:    t.ExpiresAt,
			RefreshToken: t.RefreshToken,
			TokenType:    t.TokenType,
		}
		token, err := st.qRW.UpdateOrCreateCharacterToken(ctx, arg)
		if err != nil {
			return err
		}
		ss := make([]queries.Scope, len(t.Scopes))
		for i, name := range t.Scopes {
			s, err := st.getOrCreateScope(ctx, name)
			if err != nil {
				return err
			}
			ss[i] = s
		}
		tx, err := st.dbRW.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		qtx := st.qRW.WithTx(tx)
		if err := qtx.ClearCharacterTokenScopes(ctx, int64(t.CharacterID)); err != nil {
			return err
		}
		for _, s := range ss {
			arg := queries.AddCharacterTokenScopeParams{
				CharacterTokenID: token.ID,
				ScopeID:          s.ID,
			}
			if err := qtx.AddCharacterTokenScope(ctx, arg); err != nil {
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		return fmt.Errorf("update or create token for character %d: %w", t.CharacterID, err)
	}
	return nil
}

func (st *Storage) getOrCreateScope(ctx context.Context, name string) (queries.Scope, error) {
	var s queries.Scope
	if name == "" {
		return s, fmt.Errorf("invalid scope name")
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return s, err
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	s, err = qtx.GetScope(ctx, name)
	if !errors.Is(err, sql.ErrNoRows) {
		return s, err
	} else if err != nil {
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

func characterTokenFromDBModel(o queries.CharacterToken, scopes []string) *app.CharacterToken {
	if o.CharacterID == 0 {
		panic("missing character ID")
	}
	return &app.CharacterToken{
		AccessToken:  o.AccessToken,
		CharacterID:  int32(o.CharacterID),
		ExpiresAt:    o.ExpiresAt,
		ID:           o.ID,
		RefreshToken: o.RefreshToken,
		Scopes:       scopes,
		TokenType:    o.TokenType,
	}
}
