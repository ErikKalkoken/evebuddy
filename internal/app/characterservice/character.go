// Package characterservice contains the EVE character service.
package characterservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type SSOService interface {
	Authenticate(ctx context.Context, scopes []string) (*app.Token, error)
	RefreshToken(ctx context.Context, refreshToken string) (*app.Token, error)
}

// CharacterService provides access to all managed Eve Online characters both online and from local storage.
type CharacterService struct {
	ens        *evenotification.EveNotificationService
	esiClient  *goesi.APIClient
	eus        *eveuniverseservice.EveUniverseService
	httpClient *http.Client
	scs        *statuscacheservice.StatusCacheService
	sfg        *singleflight.Group
	sso        SSOService
	st         *storage.Storage
}

type Params struct {
	EveNotificationService *evenotification.EveNotificationService
	EveUniverseService     *eveuniverseservice.EveUniverseService
	SSOService             SSOService
	StatusCacheService     *statuscacheservice.StatusCacheService
	Storage                *storage.Storage
	// optional
	HTTPClient *http.Client
	ESIClient  *goesi.APIClient
}

// New creates a new character service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(args Params) *CharacterService {
	s := &CharacterService{
		ens: args.EveNotificationService,
		eus: args.EveUniverseService,
		scs: args.StatusCacheService,
		sso: args.SSOService,
		st:  args.Storage,
		sfg: new(singleflight.Group),
	}
	if args.HTTPClient == nil {
		s.httpClient = http.DefaultClient
	} else {
		s.httpClient = args.HTTPClient
	}
	if args.ESIClient == nil {
		s.esiClient = goesi.NewAPIClient(s.httpClient, "")
	} else {
		s.esiClient = args.ESIClient
	}
	return s
}

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

// EnableTrainingWatcher enables training watcher for a character when it has an active training queue.
func (s *CharacterService) EnableTrainingWatcher(ctx context.Context, characterID int32) error {
	c, err := s.GetCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	if c.IsTrainingWatched {
		return nil
	}
	t, err := s.TotalTrainingTime(ctx, characterID)
	if err != nil {
		return err
	}
	if t.ValueOrZero() == 0 {
		return nil // no active training
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
	for id := range ids.All() {
		t, err := s.TotalTrainingTime(ctx, id)
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

func (s *CharacterService) UpdateIsTrainingWatched(ctx context.Context, id int32, v bool) error {
	return s.st.UpdateCharacterIsTrainingWatched(ctx, id, v)
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

func (s *CharacterService) ListCharactersShort(ctx context.Context) ([]*app.EntityShort[int32], error) {
	return s.st.ListCharactersShort(ctx)
}

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
func (s *CharacterService) UpdateOrCreateCharacterFromSSO(ctx context.Context, setInfo func(s string)) (int32, error) {
	ssoToken, err := s.sso.Authenticate(ctx, app.Scopes().Slice())
	if err != nil {
		return 0, err
	}
	slog.Info("Created new SSO token", "characterID", ssoToken.CharacterID, "scopes", ssoToken.Scopes)
	setInfo("Fetching character from game server. Please wait...")
	charID := ssoToken.CharacterID
	token := storage.UpdateOrCreateCharacterTokenParams{
		AccessToken:  ssoToken.AccessToken,
		CharacterID:  charID,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		Scopes:       set.Of(ssoToken.Scopes...),
		TokenType:    ssoToken.TokenType,
	}
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
	character, err := s.eus.GetOrCreateCharacterESI(ctx, token.CharacterID)
	if err != nil {
		return 0, err
	}
	err = s.st.CreateCharacter(ctx, storage.CreateCharacterParams{ID: token.CharacterID})
	if err != nil && !errors.Is(err, app.ErrAlreadyExists) {
		return 0, err
	}
	if err := s.st.UpdateOrCreateCharacterToken(ctx, token); err != nil {
		return 0, err
	}
	if err := s.scs.UpdateCharacters(ctx); err != nil {
		return 0, err
	}
	setInfo("Fetching corporation from game server. Please wait...")
	if _, err := s.eus.GetOrCreateCorporationESI(ctx, character.Corporation.ID); err != nil {
		return 0, err
	}
	if x := character.Corporation.IsNPC(); !x.IsEmpty() && !x.ValueOrZero() {
		if _, err = s.st.GetOrCreateCorporation(ctx, character.Corporation.ID); err != nil {
			return 0, err
		}
		if err := s.scs.UpdateCorporations(ctx); err != nil {
			return 0, err
		}
	}
	setInfo("Character added successfully")
	return token.CharacterID, nil
}

func (s *CharacterService) updateLocationESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionCharacterLocation {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
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
	if arg.Section != app.SectionCharacterOnline {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
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
	if arg.Section != app.SectionCharacterShip {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
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
	if arg.Section != app.SectionCharacterWalletBalance {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
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

func (s *CharacterService) addMissingEveEntitiesAndLocations(ctx context.Context, entityIDs set.Set[int32], locationIDs set.Set[int64]) error {
	g := new(errgroup.Group)
	if entityIDs.Size() > 0 {
		g.Go(func() error {
			_, err := s.eus.AddMissingEntities(ctx, entityIDs)
			return err
		})
	}
	if locationIDs.Size() > 0 {
		g.Go(func() error {
			return s.eus.AddMissingLocations(ctx, locationIDs)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}
