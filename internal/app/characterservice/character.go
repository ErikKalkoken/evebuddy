package characterservice

import (
	"context"
	"errors"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2/data/binding"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/sso"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) DeleteCharacter(ctx context.Context, id int32) error {
	if err := s.st.DeleteCharacter(ctx, id); err != nil {
		return err
	}
	slog.Info("Character deleted", "characterID", id)
	return s.scs.UpdateCharacters(ctx, s.st)
}

// EnableTrainingWatcher enables training watcher for a character when it has an active training queue.
func (s *CharacterService) EnableTrainingWatcher(ctx context.Context, characterID int32) error {
	c, err := s.GetCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	if c.IsTrainingWatched {
		return nil
	}
	t, err := s.GetTotalTrainingTime(ctx, characterID)
	if err != nil {
		return err
	}
	if t.IsEmpty() {
		return nil
	}
	err = s.UpdateIsTrainingWatched(ctx, characterID, true)
	if err != nil {
		return err
	}
	slog.Info("Enabled training watcher", "characterID", characterID)
	return nil
}

// EnableAllTrainingWatchers enables training watches for any currently active training queue.
func (s *CharacterService) EnableAllTrainingWatchers(ctx context.Context) error {
	ids, err := s.st.ListCharacterIDs(ctx)
	if err != nil {
		return err
	}
	for id := range ids.Values() {
		t, err := s.GetTotalTrainingTime(ctx, id)
		if err != nil {
			return err
		}
		if t.IsEmpty() {
			continue
		}
		err = s.UpdateIsTrainingWatched(ctx, id, true)
		if err != nil {
			return err
		}
	}
	return nil
}

// DisableAllTrainingWatchers disables training watches for all characters.
func (s *CharacterService) DisableAllTrainingWatchers(ctx context.Context) error {
	return s.st.DisableAllTrainingWatchers(ctx)
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

func (cs *CharacterService) getCharacterName(ctx context.Context, characterID int32) (string, error) {
	character, err := cs.GetCharacter(ctx, characterID)
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

func (s *CharacterService) ListCharactersShort(ctx context.Context) ([]*app.CharacterShort, error) {
	return s.st.ListCharactersShort(ctx)
}

func (s *CharacterService) UpdateIsTrainingWatched(ctx context.Context, id int32, v bool) error {
	return s.st.UpdateCharacterIsTrainingWatched(ctx, id, v)
}

// TODO: Add test for UpdateOrCreateCharacterFromSSO

// UpdateOrCreateCharacterFromSSO creates or updates a character via SSO authentication.
// The provided context is used for the SSO authentication process only and can be canceled.
func (s *CharacterService) UpdateOrCreateCharacterFromSSO(ctx context.Context, infoText binding.ExternalString) (int32, error) {
	ssoToken, err := s.sso.Authenticate(ctx, esiScopes)
	if errors.Is(err, sso.ErrAborted) {
		return 0, app.ErrAborted
	} else if err != nil {
		return 0, err
	}
	slog.Info("Created new SSO token", "characterID", ssoToken.CharacterID, "scopes", ssoToken.Scopes)
	if err := infoText.Set("Fetching character from server. Please wait..."); err != nil {
		slog.Warn("failed to set info text", "error", err)
	}
	charID := ssoToken.CharacterID
	token := app.CharacterToken{
		AccessToken:  ssoToken.AccessToken,
		CharacterID:  charID,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		Scopes:       ssoToken.Scopes,
		TokenType:    ssoToken.TokenType,
	}
	ctx = contextWithESIToken(context.Background(), token.AccessToken)
	if _, err := s.eus.GetOrCreateCharacterESI(ctx, token.CharacterID); err != nil {
		return 0, err
	}
	arg := storage.CreateCharacterParams{
		ID: token.CharacterID,
	}
	err = s.st.CreateCharacter(ctx, arg)
	if err != nil && !errors.Is(err, app.ErrAlreadyExists) {
		return 0, err
	}
	if err := s.st.UpdateOrCreateCharacterToken(ctx, &token); err != nil {
		return 0, err
	}
	if err := s.scs.UpdateCharacters(ctx, s.st); err != nil {
		return 0, err
	}
	return token.CharacterID, nil
}

func (s *CharacterService) updateLocationESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionLocation {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			location, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdLocation(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return location, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			location := data.(esi.GetCharactersCharacterIdLocationOk)
			var locationID int64
			switch {
			case location.StructureId != 0:
				locationID = location.StructureId
			case location.StationId != 0:
				locationID = int64(location.StationId)
			default:
				locationID = int64(location.SolarSystemId)
			}
			_, err := s.eus.GetOrCreateLocationESI(ctx, locationID)
			if err != nil {
				return err
			}
			if err := s.st.UpdateCharacterLocation(ctx, characterID, optional.New(locationID)); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) updateOnlineESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionOnline {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			online, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdOnline(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return online, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			online := data.(esi.GetCharactersCharacterIdOnlineOk)
			if err := s.st.UpdateCharacterLastLoginAt(ctx, characterID, optional.New(online.LastLogin)); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) updateShipESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionShip {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			ship, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdShip(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return ship, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			ship := data.(esi.GetCharactersCharacterIdShipOk)
			_, err := s.eus.GetOrCreateTypeESI(ctx, ship.ShipTypeId)
			if err != nil {
				return err
			}
			if err := s.st.UpdateCharacterShip(ctx, characterID, optional.New(ship.ShipTypeId)); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) updateWalletBalanceESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionWalletBalance {
		panic("called with wrong section")
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			balance, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWallet(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return balance, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			balance := data.(float64)
			if err := s.st.UpdateCharacterWalletBalance(ctx, characterID, optional.New(balance)); err != nil {
				return err
			}
			return nil
		})
}

// AddEveEntitiesFromSearchESI runs a search on ESI and adds the results as new EveEntity objects to the database.
// This method performs a character specific search and needs a token.
func (s *CharacterService) AddEveEntitiesFromSearchESI(ctx context.Context, characterID int32, search string) ([]int32, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return nil, err
	}
	categories := []string{
		"corporation",
		"character",
		"alliance",
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	r, _, err := s.esiClient.ESI.SearchApi.GetCharactersCharacterIdSearch(ctx, categories, characterID, search, nil)
	if err != nil {
		return nil, err
	}
	ids := slices.Concat(r.Alliance, r.Character, r.Corporation)
	missingIDs, err := s.eus.AddMissingEntities(ctx, ids)
	if err != nil {
		slog.Error("Failed to fetch missing IDs", "error", err)
		return nil, err
	}
	return missingIDs, nil
}
