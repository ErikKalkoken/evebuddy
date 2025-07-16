package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func (st *Storage) GetCharacterToken(ctx context.Context, characterID int32) (*app.CharacterToken, error) {
	t, err := st.qRO.GetCharacterToken(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("get token for character %d: %w", characterID, convertGetError(err))
	}
	rows, err := st.qRO.ListCharacterTokenScopes(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	scopes := xslices.Map(rows, func(x queries.Scope) string {
		return x.Name
	})
	t2 := characterTokenFromDBModel(t, scopes)
	return t2, nil
}

type UpdateOrCreateCharacterTokenParams struct {
	AccessToken  string
	CharacterID  int32
	ExpiresAt    time.Time
	RefreshToken string
	Scopes       []string
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
		return fmt.Errorf("updateOrCreateCharacterToken: %+v: %w", arg, err)
	}
	token, err := st.qRW.UpdateOrCreateCharacterToken(ctx, queries.UpdateOrCreateCharacterTokenParams{
		AccessToken:  arg.AccessToken,
		CharacterID:  int64(arg.CharacterID),
		ExpiresAt:    arg.ExpiresAt,
		RefreshToken: arg.RefreshToken,
		TokenType:    arg.TokenType,
	})
	if err != nil {
		return wrapErr(err)
	}
	ss := make([]queries.Scope, len(arg.Scopes))
	for i, name := range arg.Scopes {
		s, err := st.getOrCreateScope(ctx, name)
		if err != nil {
			return wrapErr(err)
		}
		ss[i] = s
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	if err := qtx.ClearCharacterTokenScopes(ctx, int64(arg.CharacterID)); err != nil {
		return wrapErr(err)
	}
	for _, s := range ss {
		arg := queries.AddCharacterTokenScopeParams{
			CharacterTokenID: token.ID,
			ScopeID:          s.ID,
		}
		if err := qtx.AddCharacterTokenScope(ctx, arg); err != nil {
			return wrapErr(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return wrapErr(err)
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

func (st *Storage) ListCharacterTokenForCorporation(ctx context.Context, corporationID int32, role app.Role) ([]*app.CharacterToken, error) {
	tokens, err := func() ([]*app.CharacterToken, error) {
		if corporationID == 0 || role == app.RoleUndefined {
			return nil, app.ErrInvalid
		}
		arg := queries.ListCharacterTokenForCorporationParams{
			CorporationID: int64(corporationID),
			Name:          role2String[role],
		}
		rows, err := st.qRO.ListCharacterTokenForCorporation(ctx, arg)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return nil, app.ErrNotFound
		}
		tokens := make([]*app.CharacterToken, 0)
		for _, r := range rows {
			ss, err := st.qRO.ListCharacterTokenScopes(ctx, r.CharacterID)
			if err != nil {
				return nil, err
			}
			scopes := xslices.Map(ss, func(x queries.Scope) string {
				return x.Name
			})
			t := characterTokenFromDBModel(r, scopes)
			tokens = append(tokens, t)
		}
		return tokens, nil
	}()
	if err != nil {
		return nil, fmt.Errorf("ListCharacterTokenForCorporation: %d %s: %w", corporationID, role.String(), err)
	}
	return tokens, nil
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
