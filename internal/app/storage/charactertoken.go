package storage

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) GetCharacterToken(ctx context.Context, characterID int64) (*app.CharacterToken, error) {
	r, err := st.qRO.GetCharacterToken(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("get token for character %d: %w", characterID, convertGetError(err))
	}
	names, err := st.qRO.ListCharacterTokenScopeNames(ctx, characterID)
	if err != nil {
		return nil, err
	}
	t2 := characterTokenFromDBModel(r, set.Of(names...))
	return t2, nil
}

type UpdateOrCreateCharacterTokenParams struct {
	AccessToken  string
	CharacterID  int64
	ExpiresAt    time.Time
	RefreshToken string
	Scopes       set.Set[string]
	TokenType    string
}

func UpdateOrCreateCharacterTokenParamsFromToken(o *app.CharacterToken) UpdateOrCreateCharacterTokenParams {
	return UpdateOrCreateCharacterTokenParams{
		AccessToken:  o.AccessToken,
		CharacterID:  o.CharacterID,
		ExpiresAt:    o.ExpiresAt,
		RefreshToken: o.RefreshToken,
		Scopes:       o.Scopes,
		TokenType:    o.TokenType,
	}
}

func (st *Storage) UpdateOrCreateCharacterToken(ctx context.Context, arg UpdateOrCreateCharacterTokenParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("updateOrCreateCharacterToken: %d %s: %w", arg.CharacterID, arg.Scopes, err)
	}
	if arg.CharacterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if arg.Scopes.Contains("") {
		return wrapErr(fmt.Errorf("invalid scope name: %w", app.ErrInvalid))
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	tokenID, err := qtx.UpdateOrCreateCharacterToken(ctx, queries.UpdateOrCreateCharacterTokenParams{
		AccessToken:  arg.AccessToken,
		CharacterID:  arg.CharacterID,
		ExpiresAt:    arg.ExpiresAt,
		RefreshToken: arg.RefreshToken,
		TokenType:    arg.TokenType,
	})
	if err != nil {
		return wrapErr(err)
	}
	if arg.Scopes.Size() > 0 {
		// create missing scopes if any
		existing, err := qtx.ListScopeNamesForNames(ctx, slices.Collect(arg.Scopes.All()))
		if err != nil {
			return wrapErr(err)
		}
		for missing := range set.Difference(arg.Scopes, set.Of(existing...)).All() {
			if err := qtx.CreateScopeIfMissing(ctx, missing); err != nil {
				return wrapErr(err)
			}
		}
	}

	names, err := qtx.ListCharacterTokenScopeNames(ctx, arg.CharacterID)
	if err != nil {
		return wrapErr(err)
	}
	current := set.Of(names...)
	if removed := set.Difference(current, arg.Scopes); removed.Size() > 0 {
		err := qtx.DeleteCharacterTokenScopes(ctx, queries.DeleteCharacterTokenScopesParams{
			CharacterTokenID: tokenID,
			Names:            slices.Collect(removed.All()),
		})
		if err != nil {
			return wrapErr(err)
		}
	}
	if added := set.Difference(arg.Scopes, current); added.Size() > 0 {
		err := qtx.AddCharacterTokenScopes(ctx, queries.AddCharacterTokenScopesParams{
			CharacterTokenID: tokenID,
			Names:            slices.Collect(added.All()),
		})
		if err != nil {
			return wrapErr(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	return nil
}

// ListCharacterTokenForCorporation returns tokens from a corporation members that match any of the provided roles an scopes.
func (st *Storage) ListCharacterTokenForCorporation(ctx context.Context, corporationID int64, roles set.Set[app.Role], scopes set.Set[string]) ([]*app.CharacterToken, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterTokenForCorporation: ID %d, roles %s, scopes %s: %w", corporationID, roles, scopes, err)
	}
	if corporationID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	var rows []queries.CharacterToken
	var err error
	if roles.Size() == 0 {
		rows, err = st.qRO.ListCharacterTokenForCorporation(ctx, corporationID)
	} else {
		rows, err = st.qRO.ListCharacterTokenForCorporationWithRoles(ctx, queries.ListCharacterTokenForCorporationWithRolesParams{
			CorporationID: corporationID,
			Roles:         slices.Collect(roles2names(roles).All()),
		})
	}
	if err != nil {
		return nil, wrapErr(err)
	}
	scopeRows, err := st.qRO.ListCharacterTokenScopesForCorporation(ctx, corporationID)
	if err != nil {
		return nil, wrapErr(err)
	}
	tokenScopes := make(map[int64]set.Set[string])
	for _, r := range scopeRows {
		s := tokenScopes[r.CharacterID]
		s.Add(r.Name)
		tokenScopes[r.CharacterID] = s
	}
	var tokens []*app.CharacterToken
	for _, r := range rows {
		scopes2, ok := tokenScopes[r.CharacterID]
		if !ok {
			scopes2 = set.Of[string]()
		}
		if !scopes2.ContainsAll(scopes.All()) {
			continue
		}
		tokens = append(tokens, characterTokenFromDBModel(r, scopes2))
	}
	return tokens, nil
}

func characterTokenFromDBModel(o queries.CharacterToken, scopes set.Set[string]) *app.CharacterToken {
	if o.CharacterID == 0 {
		panic("missing character ID")
	}
	return &app.CharacterToken{
		AccessToken:  o.AccessToken,
		CharacterID:  o.CharacterID,
		ExpiresAt:    o.ExpiresAt,
		ID:           o.ID,
		RefreshToken: o.RefreshToken,
		Scopes:       scopes,
		TokenType:    o.TokenType,
	}
}
