package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"fyne.io/fyne/v2/data/binding"

	"github.com/ErikKalkoken/evebuddy/internal/eveonline/sso"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

var ErrAborted = errors.New("process aborted prematurely")

func (s *Service) DeleteCharacter(characterID int32) error {
	return s.r.DeleteCharacter(context.Background(), characterID)
}

func (s *Service) GetCharacter(characterID int32) (*model.Character, error) {
	return s.r.GetCharacter(context.Background(), characterID)
}

func (s *Service) GetAnyCharacter() (*model.Character, error) {
	return s.r.GetFirstCharacter(context.Background())
}

func (s *Service) ListCharacters() ([]*model.Character, error) {
	return s.r.ListCharacters(context.Background())
}

func (s *Service) ListCharactersShort() ([]*model.CharacterShort, error) {
	return s.r.ListCharactersShort(context.Background())
}

// UpdateOrCreateCharacterFromSSO creates or updates a character via SSO authentication.
func (s *Service) UpdateOrCreateCharacterFromSSO(ctx context.Context, infoText binding.ExternalString) (int32, error) {
	ssoToken, err := sso.Authenticate(ctx, s.httpClient, esiScopes)
	if err != nil {
		if errors.Is(err, sso.ErrAborted) {
			return 0, ErrAborted
		}
		return 0, err
	}
	slog.Info("Created new SSO token", "characterID", ssoToken.CharacterID, "scopes", ssoToken.Scopes)
	infoText.Set("Fetching character from server. Please wait...")
	charID := ssoToken.CharacterID
	token := model.CharacterToken{
		AccessToken:  ssoToken.AccessToken,
		CharacterID:  charID,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		Scopes:       ssoToken.Scopes,
		TokenType:    ssoToken.TokenType,
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	character, err := s.getOrCreateEveCharacterESI(ctx, token.CharacterID)
	if err != nil {
		return 0, err
	}
	myCharacter := &model.Character{
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
	if err := s.r.UpdateOrCreateCharacter(ctx, arg); err != nil {
		return 0, err
	}
	if err := s.r.UpdateOrCreateCharacterToken(ctx, &token); err != nil {
		return 0, err
	}
	return token.CharacterID, nil
}

func (s *Service) updateCharacterHomeESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	clones, _, err := s.esiClient.ESI.ClonesApi.GetCharactersCharacterIdClones(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.recordCharacterSectionUpdate(ctx, characterID, model.CharacterSectionHome, clones)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	var home sql.NullInt64
	if clones.HomeLocation.LocationId != 0 {
		_, err = s.getOrCreateLocationESI(ctx, clones.HomeLocation.LocationId)
		if err != nil {
			return false, err
		}
		home.Int64 = clones.HomeLocation.LocationId
		home.Valid = true
	}
	if err := s.r.UpdateCharacterHome(ctx, characterID, home); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) updateCharacterLocationESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	location, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdLocation(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.recordCharacterSectionUpdate(ctx, characterID, model.CharacterSectionLocation, location)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	var locationID int64
	switch {
	case location.StructureId != 0:
		locationID = location.StructureId
	case location.StationId != 0:
		locationID = int64(location.StationId)
	default:
		locationID = int64(location.SolarSystemId)
	}
	_, err = s.getOrCreateLocationESI(ctx, locationID)
	if err != nil {
		return false, err
	}
	x := sql.NullInt64{Int64: locationID, Valid: true}
	if err := s.r.UpdateCharacterLocation(ctx, characterID, x); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) updateCharacterOnlineESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	online, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdOnline(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.recordCharacterSectionUpdate(ctx, characterID, model.CharacterSectionOnline, online)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	var x sql.NullTime
	if !online.LastLogin.IsZero() {
		x.Time = online.LastLogin
		x.Valid = true
	}
	if err := s.r.UpdateCharacterLastLoginAt(ctx, characterID, x); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) updateCharacterShipESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	ship, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdShip(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.recordCharacterSectionUpdate(ctx, characterID, model.CharacterSectionShip, ship)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	_, err = s.getOrCreateEveTypeESI(ctx, ship.ShipTypeId)
	if err != nil {
		return false, err
	}
	x := sql.NullInt32{Int32: ship.ShipTypeId, Valid: true}
	if err := s.r.UpdateCharacterShip(ctx, characterID, x); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) updateCharacterWalletBalanceESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithESIToken(ctx, token.AccessToken)
	balance, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWallet(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.recordCharacterSectionUpdate(ctx, characterID, model.CharacterSectionWalletBalance, balance)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	x := sql.NullFloat64{Float64: balance, Valid: true}
	if err := s.r.UpdateCharacterWalletBalance(ctx, characterID, x); err != nil {
		return false, err
	}
	return true, nil
}
