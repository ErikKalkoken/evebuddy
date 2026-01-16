package characterservice

import (
	"context"
	"errors"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

// DeleteCharacter deletes a character and corporations which have become orphaned as a result.
// It reports whether the related corporation was also deleted.
func (s *CharacterService) DeleteCharacter(ctx context.Context, id int32) (bool, error) {
	if err := s.st.DeleteCharacter(ctx, id); err != nil {
		return false, err
	}
	slog.Info("Character deleted", "characterID", id)
	if err := s.scs.UpdateCharacters(ctx); err != nil {
		return false, err
	}
	ids, err := s.st.ListOrphanedCorporationIDs(ctx)
	if err != nil {
		return false, err
	}
	if ids.Size() == 0 {
		return false, nil
	}
	for id := range ids.All() {
		err := s.st.DeleteCorporation(ctx, id)
		if err != nil {
			return false, err
		}
		slog.Info("Corporation deleted", "corporationID", id)
	}
	if err := s.scs.UpdateCorporations(ctx); err != nil {
		return false, err
	}
	return true, nil
}

// GetCharacter returns a character from storage and updates calculated fields.
func (s *CharacterService) GetCharacter(ctx context.Context, id int32) (*app.Character, error) {
	c, err := s.st.GetCharacter(ctx, id)
	if err != nil {
		return nil, err
	}
	x, err := s.calcNextCloneJump(ctx, c)
	if err != nil {
		slog.Error("get character: next clone jump", "characterID", id, "error", err)
	} else {
		c.NextCloneJump = x
	}
	return c, nil
}

func (s *CharacterService) GetAnyCharacter(ctx context.Context) (*app.Character, error) {
	return s.st.GetAnyCharacter(ctx)
}

func (s *CharacterService) getCharacterName(ctx context.Context, characterID int32) (string, error) {
	character, err := s.GetCharacter(ctx, characterID)
	if err != nil {
		return "", err
	}
	if character.EveCharacter == nil {
		return "", nil
	}
	return character.EveCharacter.Name, nil
}

func (s *CharacterService) ListCharacters(ctx context.Context) ([]*app.Character, error) {
	return s.st.ListCharacters(ctx)
}

func (s *CharacterService) ListCharacterIDs(ctx context.Context) (set.Set[int32], error) {
	return s.st.ListCharacterIDs(ctx)
}

// ListCharactersShort returns all characters in short form and ordered by name.
func (s *CharacterService) ListCharactersShort(ctx context.Context) ([]*app.EntityShort[int32], error) {
	return s.st.ListCharactersShort(ctx)
}

// ListCharacterCorporationIDs returns the corporation IDs of the characters.
func (s *CharacterService) ListCharacterCorporationIDs(ctx context.Context) (set.Set[int32], error) {
	return s.st.ListCharacterCorporationIDs(ctx)
}

// ListCharacterCorporations returns the corporations of the characters.
func (s *CharacterService) ListCharacterCorporations(ctx context.Context) ([]*app.EntityShort[int32], error) {
	return s.st.ListCharacterCorporations(ctx)
}

// HasCharacter reports whether a character exists.
func (s *CharacterService) HasCharacter(ctx context.Context, id int32) (bool, error) {
	_, err := s.GetCharacter(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// UpdateOrCreateCharacterFromSSO creates or updates a character via SSO authentication.
// The provided context is used for the SSO authentication process only and can be canceled.
// the setInfo callback is used to update info text in a dialog.
func (s *CharacterService) UpdateOrCreateCharacterFromSSO(ctx context.Context, setInfo func(s string)) (*app.Character, error) {
	ssoToken, err := s.authClient.Authorize(ctx, slices.Collect(app.Scopes().All()))
	if err != nil {
		return nil, err
	}
	slog.Info("Created new SSO token", "characterID", ssoToken.CharacterID, "scopes", ssoToken.Scopes)
	setInfo("Fetching character from game server...")
	characterID := ssoToken.CharacterID
	token := storage.UpdateOrCreateCharacterTokenParams{
		AccessToken:  ssoToken.AccessToken,
		CharacterID:  characterID,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		Scopes:       set.Of(ssoToken.Scopes...),
		TokenType:    ssoToken.TokenType,
	}
	ctx = xgoesi.NewContextWithAuth(ctx, token.CharacterID, token.AccessToken)
	character, _, err := s.eus.UpdateOrCreateCharacterESI(ctx, token.CharacterID)
	if err != nil {
		return nil, err
	}
	err = s.st.CreateCharacter(ctx, storage.CreateCharacterParams{ID: token.CharacterID})
	if err != nil && !errors.Is(err, app.ErrAlreadyExists) {
		return nil, err
	}
	if err := s.st.UpdateOrCreateCharacterToken(ctx, token); err != nil {
		return nil, err
	}
	if err := s.scs.UpdateCharacters(ctx); err != nil {
		return nil, err
	}
	setInfo("Fetching corporation from game server...")
	if _, err := s.eus.UpdateOrCreateCorporationFromESI(ctx, character.Corporation.ID); err != nil {
		return nil, err
	}
	if x := character.Corporation.IsNPC(); !x.IsEmpty() && !x.ValueOrZero() {
		if _, err = s.st.GetOrCreateCorporation(ctx, character.Corporation.ID); err != nil {
			return nil, err
		}
		if err := s.scs.UpdateCorporations(ctx); err != nil {
			return nil, err
		}
	}
	c, err := s.st.GetCharacter(ctx, characterID)
	if err != nil {
		return nil, err
	}
	return c, nil
}
