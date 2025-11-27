// Package xesi contains extensions to the esi package.
package xesi

import (
	"context"

	"github.com/antihax/goesi"
)

type contextKey string

var (
	contextCharacterID contextKey = "contextCharacterID"
)

func (c contextKey) String() string {
	return "xesi " + string(c)
}

func NewContextWithCharacterID(ctx context.Context, characterID int32) context.Context {
	return context.WithValue(ctx, contextCharacterID, characterID)
}

func CharacterIDFromContext(ctx context.Context) (int32, bool) {
	characterID, found := ctx.Value(contextCharacterID).(int32)
	if !found {
		return 0, false
	}
	return characterID, true
}

func NewContextWithAccessToken(ctx context.Context, accessToken string) context.Context {
	return context.WithValue(ctx, goesi.ContextAccessToken, accessToken)
}

func ContextHasAccessToken(ctx context.Context) bool {
	return ctx.Value(goesi.ContextAccessToken) != nil
}
