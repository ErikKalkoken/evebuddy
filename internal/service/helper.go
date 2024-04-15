package service

import (
	"context"

	"github.com/antihax/goesi"
)

func contextWithToken(ctx context.Context, accessToken string) context.Context {
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, accessToken)
	return ctx
}
