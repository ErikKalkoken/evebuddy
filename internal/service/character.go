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
	"github.com/antihax/goesi/esi"
)

var ErrAborted = errors.New("process aborted prematurely")

func (s *Service) DeleteCharacter(characterID int32) error {
	ctx := context.Background()
	if err := s.r.DeleteCharacter(ctx, characterID); err != nil {
		return err
	}
	return s.characterStatus.UpdateCharacters(ctx, s.r)
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
	if errors.Is(err, sso.ErrAborted) {
		return 0, ErrAborted
	} else if err != nil {
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
	if err := s.characterStatus.UpdateCharacters(ctx, s.r); err != nil {
		return 0, err
	}
	return token.CharacterID, nil
}

func (s *Service) updateCharacterLocationESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
	if arg.Section != model.CharacterSectionLocation {
		panic("called with wrong section")
	}
	return s.updateCharacterSectionIfChanged(
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
			_, err := s.getOrCreateLocationESI(ctx, locationID)
			if err != nil {
				return err
			}
			x := sql.NullInt64{Int64: locationID, Valid: true}
			if err := s.r.UpdateCharacterLocation(ctx, characterID, x); err != nil {
				return err
			}
			return nil
		})
}

func (s *Service) updateCharacterOnlineESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
	if arg.Section != model.CharacterSectionOnline {
		panic("called with wrong section")
	}
	return s.updateCharacterSectionIfChanged(
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
			if err := s.r.UpdateCharacterLastLoginAt(ctx, characterID, x); err != nil {
				return err
			}
			return nil
		})
}

func (s *Service) updateCharacterShipESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
	if arg.Section != model.CharacterSectionShip {
		panic("called with wrong section")
	}
	return s.updateCharacterSectionIfChanged(
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
			_, err := s.getOrCreateEveTypeESI(ctx, ship.ShipTypeId)
			if err != nil {
				return err
			}
			x := sql.NullInt32{Int32: ship.ShipTypeId, Valid: true}
			if err := s.r.UpdateCharacterShip(ctx, characterID, x); err != nil {
				return err
			}
			return nil
		})
}

func (s *Service) updateCharacterWalletBalanceESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
	if arg.Section != model.CharacterSectionWalletBalance {
		panic("called with wrong section")
	}
	return s.updateCharacterSectionIfChanged(
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
			if err := s.r.UpdateCharacterWalletBalance(ctx, characterID, x); err != nil {
				return err
			}
			return nil
		})
}

func (s *Service) CharacterStatusListStatus(characterID int32) []model.CharacterStatus {
	return s.characterStatus.ListStatus(characterID)
}

func (s *Service) CharacterStatusSummary() (float32, bool) {
	return s.characterStatus.Summary()
}

func (s *Service) CharacterStatusCharacterSummary(characterID int32) (float32, bool) {
	return s.characterStatus.CharacterSummary(characterID)
}

func (s *Service) CharacterStatusListCharacters() []*model.CharacterShort {
	return s.characterStatus.ListCharacters()
}
