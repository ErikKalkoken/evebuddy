package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2/data/binding"
	"golang.org/x/sync/errgroup"

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
func (s *Service) UpdateOrCreateMyCharacterFromSSO(ctx context.Context, infoText binding.ExternalString) error {
	ssoToken, err := sso.Authenticate(ctx, s.httpClient, esiScopes)
	if err != nil {
		if errors.Is(err, sso.ErrAborted) {
			return ErrAborted
		}
		return err
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
	_, err = s.getOrCreateEveCharacterESI(ctx, token.CharacterID)
	if err != nil {
		return err
	}
	myCharacter := &model.MyCharacter{
		ID: token.CharacterID,
	}
	if err := s.updateMyCharacterESI(ctx, myCharacter); err != nil {
		return err
	}
	arg := updateParamsFromMyCharacter(myCharacter)
	if err := s.r.UpdateOrCreateMyCharacter(ctx, arg); err != nil {
		return err
	}
	if err := s.r.UpdateOrCreateToken(ctx, &token); err != nil {
		return err
	}
	return nil
}

func (s *Service) UpdateMyCharacter(characterID int32) error {
	ctx := context.Background()
	key := fmt.Sprintf("UpdateMyCharacter-%d", characterID)
	_, err, _ := s.singleGroup.Do(key, func() (any, error) {
		err := s.updateMyCharacter(ctx, characterID)
		return struct{}{}, err
	})
	return err
}

func (s *Service) updateMyCharacter(ctx context.Context, characterID int32) error {
	token, err := s.getValidToken(ctx, characterID)
	if err != nil {
		return err
	}
	ctx = contextWithToken(ctx, token.AccessToken)
	c, err := s.r.GetMyCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	if err := s.updateMyCharacterESI(ctx, c); err != nil {
		return err
	}
	arg := updateParamsFromMyCharacter(c)
	if err := s.r.UpdateOrCreateMyCharacter(ctx, arg); err != nil {
		return err
	}
	slog.Info("Finished updating character", "characterID", characterID)
	return nil
}

// updateMyCharacterESI updates character details and related information from ESI.
func (s *Service) updateMyCharacterESI(ctx context.Context, c *model.MyCharacter) error {
	g := new(errgroup.Group)
	g.Go(func() error {
		skills, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkills(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		c.SkillPoints = sql.NullInt64{Int64: skills.TotalSp, Valid: true}
		return nil
	})
	g.Go(func() error {
		balance, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWallet(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		c.WalletBalance = sql.NullFloat64{Float64: balance, Valid: true}
		return nil
	})
	g.Go(func() error {
		online, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdOnline(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		c.LastLoginAt = sql.NullTime{Time: online.LastLogin, Valid: true}
		return nil
	})
	g.Go(func() error {
		r, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdLocation(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		location, err := s.getOrCreateEveSolarSystemESI(ctx, r.SolarSystemId)
		if err != nil {
			return err
		}
		c.Location = location
		return nil
	})
	g.Go(func() error {
		ship, _, err := s.esiClient.ESI.LocationApi.GetCharactersCharacterIdShip(ctx, c.ID, nil)
		if err != nil {
			return err
		}
		x, err := s.getOrCreateEveTypeESI(ctx, ship.ShipTypeId)
		if err != nil {
			return err
		}
		c.Ship = x
		return nil
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to update MyCharacter %d: %w", c.ID, err)
	}
	s.SectionSetUpdated(c.ID, model.UpdateSectionMyCharacter)
	return nil
}

func updateParamsFromMyCharacter(myCharacter *model.MyCharacter) storage.UpdateOrCreateMyCharacterParams {
	arg := storage.UpdateOrCreateMyCharacterParams{
		ID:            myCharacter.ID,
		LastLoginAt:   myCharacter.LastLoginAt,
		SkillPoints:   myCharacter.SkillPoints,
		WalletBalance: myCharacter.WalletBalance,
	}
	if myCharacter.Location != nil {
		arg.LocationID.Int32 = myCharacter.Location.ID
		arg.LocationID.Valid = true
	}
	if myCharacter.Ship != nil {
		arg.ShipID.Int32 = myCharacter.Ship.ID
		arg.ShipID.Valid = true
	}
	return arg
}
