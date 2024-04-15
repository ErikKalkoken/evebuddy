package repository

import (
	"context"
	"errors"
	"example/evebuddy/internal/repository/sqlc"
	"time"

	"github.com/antihax/goesi"
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

// TODO: Move into service
func (t *Token) NewContext() context.Context {
	ctx := context.WithValue(context.Background(), goesi.ContextAccessToken, t.AccessToken)
	return ctx
}

func (r *Repository) GetToken(ctx context.Context, characterID int32) (Token, error) {
	t, err := r.q.GetToken(ctx, int64(characterID))
	if err != nil {
		return Token{}, err
	}
	t2 := tokenFromDBModel(t)
	return t2, nil
}
func (r *Repository) UpdateOrCreateToken(ctx context.Context, t *Token) error {
	if t.CharacterID == 0 {
		return errors.New("can not save token without character")
	}
	arg := sqlc.CreateTokenParams{
		AccessToken:  t.AccessToken,
		CharacterID:  int64(t.CharacterID),
		ExpiresAt:    t.ExpiresAt,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
	}
	err := r.q.CreateToken(ctx, arg)
	if err != nil {
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
		if err := r.q.UpdateToken(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}
