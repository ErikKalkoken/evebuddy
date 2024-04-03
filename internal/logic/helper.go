package logic

import (
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/api/sso"
	"example/esiapp/internal/helper/set"
	"example/esiapp/internal/model"
	"fmt"
	"log/slog"
	"time"
)

// AddMissingEveEntities adds EveEntities from ESI for IDs missing in the database.
func AddMissingEveEntities(ids []int32) ([]int32, error) {
	c, err := model.FetchEveEntityIDs()
	if err != nil {
		return nil, err
	}
	current := set.NewFromSlice(c)
	incoming := set.NewFromSlice(ids)
	missing := incoming.Difference(current)

	if missing.Size() == 0 {
		return nil, nil
	}

	entities, err := esi.ResolveEntityIDs(httpClient, missing.ToSlice())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve IDs: %v %v", err, ids)
	}

	for _, entity := range entities {
		e := model.EveEntity{
			ID:       entity.ID,
			Category: model.EveEntityCategory(entity.Category),
			Name:     entity.Name,
		}
		err := e.Save()
		if err != nil {
			return nil, err
		}
	}
	slog.Debug("Added missing eve entities", "count", len(entities))
	return missing.ToSlice(), nil
}

// FetchValidToken returns a valid token for a character. Convenience function.
func FetchValidToken(characterID int32) (*model.Token, error) {
	token, err := model.FetchToken(characterID)
	if err != nil {
		return nil, err
	}
	if err := EnsureValidToken(token); err != nil {
		return nil, err
	}
	return token, nil
}

// EnsureValidToken will automatically try to refresh a token that is already or about to become invalid.
func EnsureValidToken(token *model.Token) error {
	if !token.RemainsValid(time.Second * 60) {
		slog.Debug("Need to refresh token", "characterID", token.CharacterID)
		rawToken, err := sso.RefreshToken(httpClient, token.RefreshToken)
		if err != nil {
			return err
		}
		token.AccessToken = rawToken.AccessToken
		token.RefreshToken = rawToken.RefreshToken
		token.ExpiresAt = rawToken.ExpiresAt
		err = token.Save()
		if err != nil {
			return err
		}
		slog.Info("Token refreshed", "characterID", token.CharacterID)
	}
	return nil
}
