package repository

import (
	"context"
	"database/sql"
	"errors"
	"example/evebuddy/internal/sqlc"
	"fmt"
	"time"
)

// A SSO token belonging to a character.
type Token struct {
	AccessToken  string
	CharacterID  int32
	ExpiresAt    time.Time
	RefreshToken string
	TokenType    string
}

func tokenFromDBModel(t sqlc.Token) Token {
	if t.CharacterID == 0 {
		panic("missing character ID")
	}
	return Token{
		AccessToken:  t.AccessToken,
		CharacterID:  int32(t.CharacterID),
		ExpiresAt:    t.ExpiresAt,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
	}
}

// RemainsValid reports wether a token remains valid within a duration
func (t *Token) RemainsValid(d time.Duration) bool {
	return t.ExpiresAt.After(time.Now().Add(d))
}

func (r *Repository) GetToken(ctx context.Context, characterID int32) (Token, error) {
	t, err := r.q.GetToken(ctx, int64(characterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return Token{}, fmt.Errorf("failed to get token for character %d: %w", characterID, err)
	}
	t2 := tokenFromDBModel(t)
	return t2, nil
}
func (r *Repository) UpdateOrCreateToken(ctx context.Context, t *Token) error {
	err := func() error {
		if t.CharacterID == 0 {
			return errors.New("can not save token without character")
		}
		tx, err := r.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		qtx := r.q.WithTx(tx)
		arg := sqlc.CreateTokenParams{
			AccessToken:  t.AccessToken,
			CharacterID:  int64(t.CharacterID),
			ExpiresAt:    t.ExpiresAt,
			RefreshToken: t.RefreshToken,
			TokenType:    t.TokenType,
		}
		if err := qtx.CreateToken(ctx, arg); err != nil {
			if !isSqlite3ErrConstraint(err) {
				return err
			}
			arg := sqlc.UpdateTokenParams{
				AccessToken:  t.AccessToken,
				CharacterID:  int64(t.CharacterID),
				ExpiresAt:    t.ExpiresAt,
				RefreshToken: t.RefreshToken,
				TokenType:    t.TokenType,
			}
			if err := qtx.UpdateToken(ctx, arg); err != nil {
				return err
			}
		}
		return tx.Commit()
	}()
	if err != nil {
		return fmt.Errorf("failed to update or create token for character %d: %w", t.CharacterID, err)
	}
	return nil
}
