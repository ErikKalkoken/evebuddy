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

func (s *Service) DeleteMyCharacter(characterID int32) error {
	return s.r.DeleteMyCharacter(context.Background(), characterID)
}

func (s *Service) GetMyCharacter(characterID int32) (*model.MyCharacter, error) {
	return s.r.GetMyCharacter(context.Background(), characterID)
}

func (s *Service) GetAnyMyCharacter() (*model.MyCharacter, error) {
	return s.r.GetFirstMyCharacter(context.Background())
}

func (s *Service) ListMyCharacters() ([]*model.MyCharacter, error) {
	return s.r.ListMyCharacters(context.Background())
}

func (s *Service) ListMyCharactersShort() ([]*model.MyCharacterShort, error) {
	return s.r.ListMyCharactersShort(context.Background())
}

// UpdateOrCreateMyCharacterFromSSO creates or updates a character via SSO authentication.
func (s *Service) UpdateOrCreateMyCharacterFromSSO(ctx context.Context, infoText binding.ExternalString) (int32, error) {
	ssoToken, err := sso.Authenticate(ctx, s.httpClient, esiScopes)
	if err != nil {
		if errors.Is(err, sso.ErrAborted) {
			return 0, ErrAborted
		}
		return 0, err
	}
	slog.Info("Created new SSO token", "token", ssoToken)
	infoText.Set("Fetching character from server. Please wait...")
	charID := ssoToken.CharacterID
	token := model.Token{
		AccessToken:  ssoToken.AccessToken,
		CharacterID:  charID,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		Scopes:       ssoToken.Scopes,
		TokenType:    ssoToken.TokenType,
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	character, err := s.getOrCreateEveCharacterESI(ctx, token.CharacterID)
	if err != nil {
		return 0, err
	}
	myCharacter := &model.MyCharacter{
		ID:        token.CharacterID,
		Character: character,
	}
	arg := storage.UpdateOrCreateMyCharacterParams{
		ID:            myCharacter.ID,
		LastLoginAt:   myCharacter.LastLoginAt,
		SkillPoints:   myCharacter.SkillPoints,
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
	if err := s.r.UpdateOrCreateMyCharacter(ctx, arg); err != nil {
		return 0, err
	}
	if err := s.r.UpdateOrCreateToken(ctx, &token); err != nil {
		return 0, err
	}
	return token.CharacterID, nil
}

func (s *Service) updateLocationESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	location, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdLocation(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.hasSectionChanged(ctx, characterID, model.UpdateSectionLocation, location)
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
	arg := storage.UpdateOrCreateMyCharacterParams{
		ID:         characterID,
		LocationID: sql.NullInt64{Int64: locationID, Valid: true},
	}
	if err := s.r.UpdateMyCharacter(ctx, arg); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) updateOnlineESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	online, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdOnline(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.hasSectionChanged(ctx, characterID, model.UpdateSectionOnline, online)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	arg := storage.UpdateOrCreateMyCharacterParams{
		ID:          characterID,
		LastLoginAt: sql.NullTime{Time: online.LastLogin, Valid: true},
	}
	if err := s.r.UpdateMyCharacter(ctx, arg); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) updateShipESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	ship, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdShip(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.hasSectionChanged(ctx, characterID, model.UpdateSectionShip, ship)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	x, err := s.getOrCreateEveTypeESI(ctx, ship.ShipTypeId)
	if err != nil {
		return false, err
	}
	arg := storage.UpdateOrCreateMyCharacterParams{
		ID:     characterID,
		ShipID: sql.NullInt32{Int32: x.ID, Valid: true},
	}
	if err := s.r.UpdateMyCharacter(ctx, arg); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) updateSkillsESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	skills, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkills(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.hasSectionChanged(ctx, characterID, model.UpdateSectionSkills, skills)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	arg := storage.UpdateOrCreateMyCharacterParams{
		ID:          characterID,
		SkillPoints: sql.NullInt64{Int64: skills.TotalSp, Valid: true},
	}
	if err := s.r.UpdateMyCharacter(ctx, arg); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) updateWalletBalanceESI(ctx context.Context, characterID int32) (bool, error) {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return false, err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	balance, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWallet(ctx, characterID, nil)
	if err != nil {
		return false, err
	}
	changed, err := s.hasSectionChanged(ctx, characterID, model.UpdateSectionWalletBalance, balance)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	arg := storage.UpdateOrCreateMyCharacterParams{
		ID:            characterID,
		WalletBalance: sql.NullFloat64{Float64: balance, Valid: true},
	}
	if err := s.r.UpdateMyCharacter(ctx, arg); err != nil {
		return false, err
	}
	return true, nil
}
