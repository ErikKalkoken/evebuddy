package character

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2/data/binding"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/sso"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) DeleteCharacter(ctx context.Context, characterID int32) error {
	if err := s.st.DeleteCharacter(ctx, characterID); err != nil {
		return err
	}
	return s.cs.UpdateCharacters(ctx, s.st)
}

func (s *CharacterService) GetCharacter(ctx context.Context, characterID int32) (*app.Character, error) {
	o, err := s.st.GetCharacter(ctx, characterID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	return o, err
}

func (s *CharacterService) GetAnyCharacter(ctx context.Context) (*app.Character, error) {
	o, err := s.st.GetFirstCharacter(ctx)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	return o, err
}

func (s *CharacterService) ListCharacters(ctx context.Context) ([]*app.Character, error) {
	return s.st.ListCharacters(ctx)
}

func (s *CharacterService) ListCharactersShort(ctx context.Context) ([]*app.CharacterShort, error) {
	return s.st.ListCharactersShort(ctx)
}

// UpdateOrCreateCharacterFromSSO creates or updates a character via SSO authentication.
func (s *CharacterService) UpdateOrCreateCharacterFromSSO(ctx context.Context, infoText binding.ExternalString) (int32, error) {
	ssoToken, err := sso.Authenticate(ctx, s.httpClient, esiScopes)
	if errors.Is(err, sso.ErrAborted) {
		return 0, ErrAborted
	} else if err != nil {
		return 0, err
	}
	slog.Info("Created new SSO token", "characterID", ssoToken.CharacterID, "scopes", ssoToken.Scopes)
	infoText.Set("Fetching character from server. Please wait...")
	charID := ssoToken.CharacterID
	token := app.CharacterToken{
		AccessToken:  ssoToken.AccessToken,
		CharacterID:  charID,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		Scopes:       ssoToken.Scopes,
		TokenType:    ssoToken.TokenType,
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	character, err := s.eu.GetOrCreateEveCharacterESI(ctx, token.CharacterID)
	if err != nil {
		return 0, err
	}
	myCharacter := &app.Character{
		ID:           token.CharacterID,
		EveCharacter: character,
	}
	arg := storage.UpdateOrCreateCharacterParams{
		ID:            myCharacter.ID,
		LastLoginAt:   myCharacter.LastLoginAt,
		TotalSP:       myCharacter.TotalSP,
		WalletBalance: myCharacter.WalletBalance,
	}
	if myCharacter.Location != nil {
		arg.LocationID.Int64 = myCharacter.Location.ID
		arg.LocationID.Valid = true
	}
	if myCharacter.Ship != nil {
		arg.ShipID.Int32 = myCharacter.Ship.ID
		arg.ShipID.Valid = true
	}
	if err := s.st.UpdateOrCreateCharacter(ctx, arg); err != nil {
		return 0, err
	}
	if err := s.st.UpdateOrCreateCharacterToken(ctx, &token); err != nil {
		return 0, err
	}
	if err := s.cs.UpdateCharacters(ctx, s.st); err != nil {
		return 0, err
	}
	return token.CharacterID, nil
}

func (s *CharacterService) updateCharacterLocationESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
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
			_, err := s.eu.GetOrCreateEveLocationESI(ctx, locationID)
			if err != nil {
				return err
			}
			x := sql.NullInt64{Int64: locationID, Valid: true}
			if err := s.st.UpdateCharacterLocation(ctx, characterID, x); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) updateCharacterOnlineESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
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
			var x sql.NullTime
			if !online.LastLogin.IsZero() {
				x.Time = online.LastLogin
				x.Valid = true
			}
			if err := s.st.UpdateCharacterLastLoginAt(ctx, characterID, x); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) updateCharacterShipESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
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
			_, err := s.eu.GetOrCreateEveTypeESI(ctx, ship.ShipTypeId)
			if err != nil {
				return err
			}
			x := sql.NullInt32{Int32: ship.ShipTypeId, Valid: true}
			if err := s.st.UpdateCharacterShip(ctx, characterID, x); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) updateCharacterWalletBalanceESI(ctx context.Context, arg UpdateSectionParams) (bool, error) {
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
			x := sql.NullFloat64{Float64: balance, Valid: true}
			if err := s.st.UpdateCharacterWalletBalance(ctx, characterID, x); err != nil {
				return err
			}
			return nil
		})
}

// AddEveEntitiesFromCharacterSearchESI runs a search on ESI and adds the results as new EveEntity objects to the database.
// This method performs a character specific search and needs a token.
func (s *CharacterService) AddEveEntitiesFromCharacterSearchESI(ctx context.Context, characterID int32, search string) ([]int32, error) {
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
	missingIDs, err := s.eu.AddMissingEveEntities(ctx, ids)
	if err != nil {
		slog.Error("Failed to fetch missing IDs", "error", err)
		return nil, err
	}
	return missingIDs, nil
}
