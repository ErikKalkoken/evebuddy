package logic

import (
	"context"
	"errors"
	"example/evebuddy/internal/api/sso"
	"example/evebuddy/internal/model"
	"log/slog"
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

func tokenFromDBModel(t model.Token) Token {
	if t.CharacterID == 0 {
		panic("missing character ID")
	}
	return Token{
		AccessToken:  t.AccessToken,
		CharacterID:  t.CharacterID,
		ExpiresAt:    t.ExpiresAt,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
	}
}

func tokenDBModelFromToken(t Token) model.Token {
	return model.Token{
		AccessToken:  t.AccessToken,
		CharacterID:  t.CharacterID,
		ExpiresAt:    t.ExpiresAt,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
	}
}

// RemainsValid reports wether a token remains valid within a duration
func (t *Token) RemainsValid(d time.Duration) bool {
	return t.ExpiresAt.After(time.Now().Add(d))
}

func (t *Token) Save() error {
	if t.CharacterID == 0 {
		return errors.New("can not save token without character")
	}
	t2 := tokenDBModelFromToken(*t)
	err := t2.Save()
	return err
}

// ensureValid will automatically try to refresh a token that is already or about to become invalid.
func (t *Token) ensureValid() error {
	if !t.RemainsValid(time.Second * 60) {
		slog.Debug("Need to refresh token", "characterID", t.CharacterID)
		rawToken, err := sso.RefreshToken(httpClient, t.RefreshToken)
		if err != nil {
			return err
		}
		t.AccessToken = rawToken.AccessToken
		t.RefreshToken = rawToken.RefreshToken
		t.ExpiresAt = rawToken.ExpiresAt
		err = t.Save()
		if err != nil {
			return err
		}
		slog.Info("Token refreshed", "characterID", t.CharacterID)
	}
	return nil
}

// getValidToken returns a valid token for a character. Convenience function.
func getValidToken(characterID int32) (*Token, error) {
	t, err := model.GetToken(characterID)
	if err != nil {
		return nil, err
	}
	t2 := tokenFromDBModel(t)
	if err := t2.ensureValid(); err != nil {
		return nil, err
	}
	return &t2, nil
}

func (token *Token) newContext() context.Context {
	ctx := context.WithValue(context.Background(), goesi.ContextAccessToken, token.AccessToken)
	return ctx
}
