package characters

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"fyne.io/fyne/v2/data/binding"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/eveonline/sso"
	"github.com/ErikKalkoken/evebuddy/internal/helper/cache"
	igoesi "github.com/ErikKalkoken/evebuddy/internal/helper/goesi"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service/characterstatus"
	"github.com/ErikKalkoken/evebuddy/internal/service/dictionary"
	"github.com/ErikKalkoken/evebuddy/internal/service/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
)

var ErrAborted = errors.New("process aborted prematurely")

type Characters struct {
	esiClient       *goesi.APIClient
	httpClient      *http.Client
	r               *storage.Storage
	singleGroup     *singleflight.Group
	EveUniverse     *eveuniverse.EveUniverse
	Dictionary      *dictionary.Dictionary
	CharacterStatus *characterstatus.CharacterStatusCache
}

// New creates a new Characters service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(
	r *storage.Storage,
	httpClient *http.Client,
	esiClient *goesi.APIClient,
	cs *characterstatus.CharacterStatusCache,
	dt *dictionary.Dictionary,
	eu *eveuniverse.EveUniverse,
) *Characters {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if esiClient == nil {
		esiClient = goesi.NewAPIClient(httpClient, "")
	}
	if cs == nil {
		cache := cache.New()
		cs = characterstatus.New(cache)
	}
	if dt == nil {
		dt = dictionary.New(r)
	}
	if eu == nil {
		eu = eveuniverse.New(r, esiClient)
	}
	x := &Characters{
		r:               r,
		esiClient:       esiClient,
		httpClient:      httpClient,
		singleGroup:     new(singleflight.Group),
		EveUniverse:     eu,
		CharacterStatus: cs,
		Dictionary:      dt,
	}
	return x
}

func (s *Characters) DeleteCharacter(ctx context.Context, characterID int32) error {
	if err := s.r.DeleteCharacter(ctx, characterID); err != nil {
		return err
	}
	return s.CharacterStatus.UpdateCharacters(ctx, s.r)
}

func (s *Characters) GetCharacter(ctx context.Context, characterID int32) (*model.Character, error) {
	return s.r.GetCharacter(ctx, characterID)
}

func (s *Characters) GetAnyCharacter(ctx context.Context) (*model.Character, error) {
	return s.r.GetFirstCharacter(ctx)
}

func (s *Characters) ListCharacters() ([]*model.Character, error) {
	return s.r.ListCharacters(context.Background())
}

func (s *Characters) ListCharactersShort(ctx context.Context) ([]*model.CharacterShort, error) {
	return s.r.ListCharactersShort(ctx)
}

// UpdateOrCreateCharacterFromSSO creates or updates a character via SSO authentication.
func (s *Characters) UpdateOrCreateCharacterFromSSO(ctx context.Context, infoText binding.ExternalString) (int32, error) {
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
	ctx = igoesi.ContextWithESIToken(ctx, token.AccessToken)
	character, err := s.EveUniverse.GetOrCreateEveCharacterESI(ctx, token.CharacterID)
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
	if err := s.CharacterStatus.UpdateCharacters(ctx, s.r); err != nil {
		return 0, err
	}
	return token.CharacterID, nil
}

func (s *Characters) updateCharacterLocationESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
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
			_, err := s.EveUniverse.GetOrCreateLocationESI(ctx, locationID)
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

func (s *Characters) updateCharacterOnlineESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
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

func (s *Characters) updateCharacterShipESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
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
			_, err := s.EveUniverse.GetOrCreateEveTypeESI(ctx, ship.ShipTypeId)
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

func (s *Characters) updateCharacterWalletBalanceESI(ctx context.Context, arg UpdateCharacterSectionParams) (bool, error) {
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

// AddEveEntitiesFromCharacterSearchESI runs a search on ESI and adds the results as new EveEntity objects to the database.
// This method performs a character specific search and needs a token.
func (s *Characters) AddEveEntitiesFromCharacterSearchESI(ctx context.Context, characterID int32, search string) ([]int32, error) {
	token, err := s.getValidCharacterToken(ctx, characterID)
	if err != nil {
		return nil, err
	}
	categories := []string{
		"corporation",
		"character",
		"alliance",
	}
	ctx = igoesi.ContextWithESIToken(ctx, token.AccessToken)
	r, _, err := s.esiClient.ESI.SearchApi.GetCharactersCharacterIdSearch(ctx, categories, characterID, search, nil)
	if err != nil {
		return nil, err
	}
	ids := slices.Concat(r.Alliance, r.Character, r.Corporation)
	missingIDs, err := s.EveUniverse.AddMissingEveEntities(ctx, ids)
	if err != nil {
		slog.Error("Failed to fetch missing IDs", "error", err)
		return nil, err
	}
	return missingIDs, nil
}
