// Package characterservice contains the EVE character service.
package characterservice

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	esioptional "github.com/antihax/goesi/optional"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

var esiScopes = []string{
	"esi-assets.read_assets.v1",
	"esi-characters.read_contacts.v1",
	"esi-characters.read_corporation_roles.v1",
	"esi-characters.read_notifications.v1",
	"esi-clones.read_clones.v1",
	"esi-clones.read_implants.v1",
	"esi-contracts.read_character_contracts.v1",
	"esi-industry.read_character_jobs.v1",
	"esi-industry.read_corporation_jobs.v1",
	"esi-location.read_location.v1",
	"esi-location.read_online.v1",
	"esi-location.read_ship_type.v1",
	"esi-mail.organize_mail.v1",
	"esi-mail.read_mail.v1",
	"esi-mail.send_mail.v1",
	"esi-planets.manage_planets.v1",
	"esi-search.search_structures.v1",
	"esi-skills.read_skillqueue.v1",
	"esi-skills.read_skills.v1",
	"esi-universe.read_structures.v1",
	"esi-wallet.read_character_wallet.v1",
}

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
	HttpClient *http.Client
	EsiClient  *goesi.APIClient
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
	if args.HttpClient == nil {
		s.httpClient = http.DefaultClient
	} else {
		s.httpClient = args.HttpClient
	}
	if args.EsiClient == nil {
		s.esiClient = goesi.NewAPIClient(s.httpClient, "")
	} else {
		s.esiClient = args.EsiClient
	}
	return s
}

const (
	assetNamesMaxIDs = 999
)

func (s *CharacterService) ListAssetsInShipHangar(ctx context.Context, characterID int32, locationID int64) ([]*app.CharacterAsset, error) {
	return s.st.ListCharacterAssetsInShipHangar(ctx, characterID, locationID)
}

func (s *CharacterService) ListAssetsInItemHangar(ctx context.Context, characterID int32, locationID int64) ([]*app.CharacterAsset, error) {
	return s.st.ListCharacterAssetsInItemHangar(ctx, characterID, locationID)
}

func (s *CharacterService) ListAssetsInLocation(ctx context.Context, characterID int32, locationID int64) ([]*app.CharacterAsset, error) {
	return s.st.ListCharacterAssetsInLocation(ctx, characterID, locationID)
}

func (s *CharacterService) ListAssets(ctx context.Context, characterID int32) ([]*app.CharacterAsset, error) {
	return s.st.ListCharacterAssets(ctx, characterID)
}

func (s *CharacterService) ListAllAssets(ctx context.Context) ([]*app.CharacterAsset, error) {
	return s.st.ListAllCharacterAssets(ctx)
}

type esiCharacterAssetPlus struct {
	esi.GetCharactersCharacterIdAssets200Ok
	Name string
}

func (s *CharacterService) updateAssetsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionAssets {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	hasChanged, err := s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			assets, err := fetchFromESIWithPaging(
				func(pageNum int) ([]esi.GetCharactersCharacterIdAssets200Ok, *http.Response, error) {
					arg := &esi.GetCharactersCharacterIdAssetsOpts{
						Page: esioptional.NewInt32(int32(pageNum)),
					}
					return s.esiClient.ESI.AssetsApi.GetCharactersCharacterIdAssets(ctx, characterID, arg)
				})
			if err != nil {
				return false, err
			}
			slog.Debug("Received assets from ESI", "count", len(assets), "characterID", characterID)
			ids := make([]int64, len(assets))
			for i, a := range assets {
				ids[i] = a.ItemId
			}
			names, err := s.fetchAssetNamesESI(ctx, characterID, ids)
			if err != nil {
				return false, err
			}
			slog.Debug("Received asset names from ESI", "count", len(names), "characterID", characterID)
			assetsPlus := make([]esiCharacterAssetPlus, len(assets))
			for i, a := range assets {
				o := esiCharacterAssetPlus{
					GetCharactersCharacterIdAssets200Ok: a,
					Name:                                names[a.ItemId],
				}
				assetsPlus[i] = o
			}
			return assetsPlus, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			assets := data.([]esiCharacterAssetPlus)
			incomingIDs := set.Of[int64]()
			for _, ca := range assets {
				incomingIDs.Add(ca.ItemId)
			}
			typeIDs := set.Of[int32]()
			locationIDs := set.Of[int64]()
			for _, ca := range assets {
				typeIDs.Add(ca.TypeId)
				if !incomingIDs.Contains(ca.LocationId) {
					locationIDs.Add(ca.LocationId) // location IDs that are not referencing other itemIDs are locations
				}
			}
			g := new(errgroup.Group)
			g.Go(func() error {
				return s.eus.AddMissingLocations(ctx, locationIDs)
			})
			g.Go(func() error {
				return s.eus.AddMissingTypes(ctx, typeIDs)
			})
			if err := g.Wait(); err != nil {
				return err
			}
			currentIDs, err := s.st.ListCharacterAssetIDs(ctx, characterID)
			if err != nil {
				return err
			}
			var updated, created int
			for _, a := range assets {
				if currentIDs.Contains(a.ItemId) {
					arg := storage.UpdateCharacterAssetParams{
						CharacterID:  characterID,
						ItemID:       a.ItemId,
						LocationFlag: a.LocationFlag,
						LocationID:   a.LocationId,
						LocationType: a.LocationType,
						Name:         a.Name,
						Quantity:     a.Quantity,
					}
					if err := s.st.UpdateCharacterAsset(ctx, arg); err != nil {
						return err
					}
					updated++
				} else {
					arg := storage.CreateCharacterAssetParams{
						CharacterID:     characterID,
						EveTypeID:       a.TypeId,
						IsBlueprintCopy: a.IsBlueprintCopy,
						IsSingleton:     a.IsSingleton,
						ItemID:          a.ItemId,
						LocationFlag:    a.LocationFlag,
						LocationID:      a.LocationId,
						LocationType:    a.LocationType,
						Name:            a.Name,
						Quantity:        a.Quantity,
					}
					if err := s.st.CreateCharacterAsset(ctx, arg); err != nil {
						return err
					}
					created++
				}
			}
			if _, err := s.UpdateAssetTotalValue(ctx, characterID); err != nil {
				return err
			}
			slog.Info("Stored character assets", "characterID", characterID, "created", created, "updated", updated)
			if ids := set.Difference(currentIDs, incomingIDs); ids.Size() > 0 {
				if err := s.st.DeleteCharacterAssets(ctx, characterID, ids.Slice()); err != nil {
					return err
				}
				slog.Info("Deleted obsolete character assets", "characterID", characterID, "count", ids.Size())
			}
			return nil
		})
	if err != nil {
		return false, err
	}
	_, err = s.UpdateAssetTotalValue(ctx, arg.CharacterID)
	if err != nil {
		slog.Error("Failed to update asset total value", "characterID", arg.CharacterID, "err", err)
	}
	return hasChanged, err
}

func (s *CharacterService) fetchAssetNamesESI(ctx context.Context, characterID int32, ids []int64) (map[int64]string, error) {
	numResults := len(ids) / assetNamesMaxIDs
	if len(ids)%assetNamesMaxIDs > 0 {
		numResults++
	}
	results := make([][]esi.PostCharactersCharacterIdAssetsNames200Ok, numResults)
	g := new(errgroup.Group)
	for num, chunk := range xiter.Count(slices.Chunk(ids, assetNamesMaxIDs), 0) {
		g.Go(func() error {
			names, _, err := s.esiClient.ESI.AssetsApi.PostCharactersCharacterIdAssetsNames(ctx, characterID, chunk, nil)
			if err != nil {
				return err
			}
			results[num] = names
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		// We can live temporarily without asset names and will try again to fetch them next time
		// If some of the requests have succeeded we will use those names
		slog.Warn("Failed to fetch asset names", "characterID", characterID, "err", err)
	}
	m := make(map[int64]string)
	for _, names := range results {
		for _, n := range names {
			if n.Name != "None" {
				m[n.ItemId] = n.Name
			}
		}
	}
	return m, nil
}

func (s *CharacterService) AssetTotalValue(ctx context.Context, characterID int32) (optional.Optional[float64], error) {
	return s.st.GetCharacterAssetValue(ctx, characterID)
}

func (s *CharacterService) UpdateAssetTotalValue(ctx context.Context, characterID int32) (float64, error) {
	v, err := s.st.CalculateCharacterAssetTotalValue(ctx, characterID)
	if err != nil {
		return 0, err
	}
	if err := s.st.UpdateCharacterAssetValue(ctx, characterID, optional.From(v)); err != nil {
		return 0, err
	}
	return v, nil
}

func (s *CharacterService) GetAttributes(ctx context.Context, characterID int32) (*app.CharacterAttributes, error) {
	return s.st.GetCharacterAttributes(ctx, characterID)
}

func (s *CharacterService) updateAttributesESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionAttributes {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			attributes, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdAttributes(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return attributes, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			attributes := data.(esi.GetCharactersCharacterIdAttributesOk)
			arg := storage.UpdateOrCreateCharacterAttributesParams{
				CharacterID:   characterID,
				BonusRemaps:   int(attributes.BonusRemaps),
				Charisma:      int(attributes.Charisma),
				Intelligence:  int(attributes.Intelligence),
				LastRemapDate: attributes.LastRemapDate,
				Memory:        int(attributes.Memory),
				Perception:    int(attributes.Perception),
				Willpower:     int(attributes.Willpower),
			}
			if err := s.st.UpdateOrCreateCharacterAttributes(ctx, arg); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) DeleteCharacter(ctx context.Context, id int32) error {
	if err := s.st.DeleteCharacter(ctx, id); err != nil {
		return err
	}
	slog.Info("Character deleted", "characterID", id)
	if err := s.scs.UpdateCharacters(ctx); err != nil {
		return err
	}
	ids, err := s.st.ListOrphanedCorporationIDs(ctx)
	if err != nil {
		return err
	}
	if ids.Size() == 0 {
		return nil
	}
	for id := range ids.All() {
		err := s.st.DeleteCorporation(ctx, id)
		if err != nil {
			return nil
		}
		slog.Info("Corporation deleted", "corporationID", id)
	}
	return s.scs.UpdateCorporations(ctx)
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
	for id := range ids.All() {
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

func (s *CharacterService) UpdateIsTrainingWatched(ctx context.Context, id int32, v bool) error {
	return s.st.UpdateCharacterIsTrainingWatched(ctx, id, v)
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
func (s *CharacterService) UpdateOrCreateCharacterFromSSO(ctx context.Context, infoText binding.ExternalString) (int32, error) {
	ssoToken, err := s.sso.Authenticate(ctx, esiScopes)
	if err != nil {
		return 0, err
	}
	slog.Info("Created new SSO token", "characterID", ssoToken.CharacterID, "scopes", ssoToken.Scopes)
	if err := infoText.Set("Fetching character from game server. Please wait..."); err != nil {
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
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
	character, err := s.eus.GetOrCreateCharacterESI(ctx, token.CharacterID)
	if err != nil {
		return 0, err
	}
	err = s.st.CreateCharacter(ctx, storage.CreateCharacterParams{ID: token.CharacterID})
	if err != nil && !errors.Is(err, app.ErrAlreadyExists) {
		return 0, err
	}
	if err := s.st.UpdateOrCreateCharacterToken(ctx, &token); err != nil {
		return 0, err
	}
	if err := s.scs.UpdateCharacters(ctx); err != nil {
		return 0, err
	}
	if x := character.Corporation.IsNPC(); !x.IsEmpty() && !x.ValueOrZero() {
		if err := infoText.Set("Fetching corporation from game server. Please wait..."); err != nil {
			slog.Warn("failed to set info text", "error", err)
		}
		if _, err := s.eus.GetOrCreateCorporationESI(ctx, character.Corporation.ID); err != nil {
			return 0, err
		}
		if _, err = s.st.GetOrCreateCorporation(ctx, character.Corporation.ID); err != nil {
			return 0, err
		}
		if err := s.scs.UpdateCorporations(ctx); err != nil {
			return 0, err
		}
	}
	if err := infoText.Set("Character added successfully"); err != nil {
		slog.Warn("failed to set info text", "error", err)
	}
	return token.CharacterID, nil
}

func (s *CharacterService) updateLocationESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionLocation {
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
			if err := s.st.UpdateCharacterLocation(ctx, characterID, optional.From(locationID)); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) updateOnlineESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionOnline {
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
			if err := s.st.UpdateCharacterLastLoginAt(ctx, characterID, optional.From(online.LastLogin)); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) updateShipESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionShip {
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
			if err := s.st.UpdateCharacterShip(ctx, characterID, optional.From(ship.ShipTypeId)); err != nil {
				return err
			}
			return nil
		})
}

func (s *CharacterService) updateWalletBalanceESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionWalletBalance {
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
			if err := s.st.UpdateCharacterWalletBalance(ctx, characterID, optional.From(balance)); err != nil {
				return err
			}
			return nil
		})
}

// AddEveEntitiesFromSearchESI runs a search on ESI and adds the results as new EveEntity objects to the database.
// This method performs a character specific search and needs a token.
func (s *CharacterService) AddEveEntitiesFromSearchESI(ctx context.Context, characterID int32, search string) ([]int32, error) {
	token, err := s.GetValidCharacterToken(ctx, characterID)
	if err != nil {
		return nil, err
	}
	categories := []string{
		"corporation",
		"character",
		"alliance",
	}
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
	r, _, err := s.esiClient.ESI.SearchApi.GetCharactersCharacterIdSearch(ctx, categories, characterID, search, nil)
	if err != nil {
		return nil, err
	}
	ids := set.Union(set.Of(r.Alliance...), set.Of(r.Character...), set.Of(r.Corporation...))
	missingIDs, err := s.eus.AddMissingEntities(ctx, ids)
	if err != nil {
		slog.Error("Failed to fetch missing IDs", "error", err)
		return nil, err
	}
	return missingIDs.Slice(), nil
}

// GetContract fetches and returns a contract from the database.
func (s *CharacterService) GetContract(ctx context.Context, characterID, contractID int32) (*app.CharacterContract, error) {
	return s.st.GetCharacterContract(ctx, characterID, contractID)
}

func (s *CharacterService) CountContractBids(ctx context.Context, contractID int64) (int, error) {
	x, err := s.st.ListCharacterContractBidIDs(ctx, contractID)
	if err != nil {
		return 0, err
	}
	return x.Size(), nil
}

func (s *CharacterService) GetContractTopBid(ctx context.Context, contractID int64) (*app.CharacterContractBid, error) {
	bids, err := s.st.ListCharacterContractBids(ctx, contractID)
	if err != nil {
		return nil, err
	}
	if len(bids) == 0 {
		return nil, app.ErrNotFound
	}
	var max float32
	var top *app.CharacterContractBid
	for _, b := range bids {
		if top == nil || b.Amount > max {
			top = b
		}
	}
	return top, nil
}

func (s *CharacterService) NotifyUpdatedContracts(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error {
	cc, err := s.st.ListCharacterContractsForNotify(ctx, characterID, earliest)
	if err != nil {
		return err
	}
	characterName, err := s.getCharacterName(ctx, characterID)
	if err != nil {
		return err
	}
	for _, c := range cc {
		if c.Status == c.StatusNotified {
			continue
		}
		if c.Acceptor != nil && c.Acceptor.ID == characterID {
			continue // ignore status changed caused by the current character
		}
		var content string
		name := "'" + c.NameDisplay() + "'"
		switch c.Type {
		case app.ContractTypeCourier:
			switch c.Status {
			case app.ContractStatusInProgress:
				content = fmt.Sprintf("Contract %s has been accepted by %s", name, c.ContractorDisplay())
			case app.ContractStatusFinished:
				content = fmt.Sprintf("Contract %s has been delivered", name)
			case app.ContractStatusFailed:
				content = fmt.Sprintf("Contract %s has been failed by %s", name, c.ContractorDisplay())
			}
		case app.ContractTypeItemExchange:
			switch c.Status {
			case app.ContractStatusFinished:
				content = fmt.Sprintf("Contract %s has been accepted by %s", name, c.ContractorDisplay())
			}
		}
		if content == "" {
			continue
		}
		title := fmt.Sprintf("%s: Contract updated", characterName)
		notify(title, content)
		if err := s.st.UpdateCharacterContractNotified(ctx, c.ID, c.Status); err != nil {
			return fmt.Errorf("record contract notification: %w", err)
		}
	}
	return nil
}

func (s *CharacterService) ListAllContracts(ctx context.Context) ([]*app.CharacterContract, error) {
	return s.st.ListAllCharacterContracts(ctx)
}

func (s *CharacterService) ListContractItems(ctx context.Context, contractID int64) ([]*app.CharacterContractItem, error) {
	return s.st.ListCharacterContractItems(ctx, contractID)
}

var contractAvailabilityFromESIValue = map[string]app.ContractAvailability{
	"alliance":    app.ContractAvailabilityAlliance,
	"corporation": app.ContractAvailabilityCorporation,
	"personal":    app.ContractAvailabilityPersonal,
	"public":      app.ContractAvailabilityPublic,
}

var contractStatusFromESIValue = map[string]app.ContractStatus{
	"cancelled":           app.ContractStatusCancelled,
	"deleted":             app.ContractStatusDeleted,
	"failed":              app.ContractStatusFailed,
	"finished_contractor": app.ContractStatusFinishedContractor,
	"finished_issuer":     app.ContractStatusFinishedIssuer,
	"finished":            app.ContractStatusFinished,
	"in_progress":         app.ContractStatusInProgress,
	"outstanding":         app.ContractStatusOutstanding,
	"rejected":            app.ContractStatusRejected,
	"reversed":            app.ContractStatusReversed,
}

var contractTypeFromESIValue = map[string]app.ContractType{
	"auction":       app.ContractTypeAuction,
	"courier":       app.ContractTypeCourier,
	"item_exchange": app.ContractTypeItemExchange,
	"loan":          app.ContractTypeLoan,
	"unknown":       app.ContractTypeUnknown,
}

// updateContractsESI updates the wallet journal from ESI and reports whether it has changed.
func (s *CharacterService) updateContractsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionContracts {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			contracts, err := fetchFromESIWithPaging(
				func(pageNum int) ([]esi.GetCharactersCharacterIdContracts200Ok, *http.Response, error) {
					arg := &esi.GetCharactersCharacterIdContractsOpts{
						Page: esioptional.NewInt32(int32(pageNum)),
					}
					return s.esiClient.ESI.ContractsApi.GetCharactersCharacterIdContracts(ctx, characterID, arg)
				})
			if err != nil {
				return false, err
			}
			slog.Debug("Received contracts from ESI", "characterID", characterID, "count", len(contracts))
			return contracts, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			contracts := data.([]esi.GetCharactersCharacterIdContracts200Ok)
			// fetch missing eve entities
			var entityIDs set.Set[int32]
			var locationIDs set.Set[int64]
			for _, c := range contracts {
				entityIDs.Add(c.IssuerId)
				entityIDs.Add(c.IssuerCorporationId)
				if c.AcceptorId != 0 {
					entityIDs.Add(c.AcceptorId)
				}
				if c.AssigneeId != 0 {
					entityIDs.Add(c.AssigneeId)
				}
				if c.StartLocationId != 0 {
					locationIDs.Add(c.StartLocationId)
				}
				if c.EndLocationId != 0 {
					locationIDs.Add(c.EndLocationId)
				}
			}
			g := new(errgroup.Group)
			g.Go(func() error {
				_, err := s.eus.AddMissingEntities(ctx, entityIDs)
				return err
			})
			g.Go(func() error {
				return s.eus.AddMissingLocations(ctx, locationIDs)
			})
			if err := g.Wait(); err != nil {
				return err
			}
			if err := s.eus.AddMissingLocations(ctx, locationIDs); err != nil {
				return err
			}
			// identify new contracts
			ii, err := s.st.ListCharacterContractIDs(ctx, characterID)
			if err != nil {
				return err
			}
			existingIDs := set.Of(ii...)
			var existingContracts, newContracts []esi.GetCharactersCharacterIdContracts200Ok
			for _, c := range contracts {
				if existingIDs.Contains(c.ContractId) {
					existingContracts = append(existingContracts, c)
				} else {
					newContracts = append(newContracts, c)
				}
			}
			slog.Debug("contracts", "existing", existingIDs, "entries", contracts)
			// create new entries
			if len(newContracts) > 0 {
				var count int
				for _, c := range newContracts {
					if err := s.createNewContract(ctx, characterID, c); err != nil {
						slog.Error("create contract", "contract", c, "error", err)
						continue
					} else {
						count++
					}
				}
				slog.Info("Stored new contracts", "characterID", characterID, "count", count)
			}
			if len(existingContracts) > 0 {
				var count int
				for _, c := range existingContracts {
					if err := s.updateContract(ctx, characterID, c); err != nil {
						slog.Error("update contract", "contract", c, "error", err)
						continue
					} else {
						count++
					}
				}
				slog.Info("Updated contracts", "characterID", characterID, "count", count)
			}
			// add new bids for auctions
			for _, c := range contracts {
				if c.Type_ != "auction" {
					continue
				}
				err := s.updateContractBids(ctx, characterID, c.ContractId)
				if err != nil {
					slog.Error("update contract bids", "contract", c, "error", err)
					continue
				}
			}
			return nil
		})
}

func (s *CharacterService) createNewContract(ctx context.Context, characterID int32, c esi.GetCharactersCharacterIdContracts200Ok) error {
	availability, ok := contractAvailabilityFromESIValue[c.Availability]
	if !ok {
		return fmt.Errorf("unknown availability: %s", c.Availability)
	}
	status, ok := contractStatusFromESIValue[c.Status]
	if !ok {
		return fmt.Errorf("unknown status: %s", c.Status)
	}
	typ, ok := contractTypeFromESIValue[c.Type_]
	if !ok {
		return fmt.Errorf("unknown type: %s", c.Type_)
	}
	arg := storage.CreateCharacterContractParams{
		AcceptorID:          c.AcceptorId,
		AssigneeID:          c.AssigneeId,
		Availability:        availability,
		Buyout:              c.Buyout,
		CharacterID:         characterID,
		Collateral:          c.Collateral,
		ContractID:          c.ContractId,
		DateAccepted:        c.DateAccepted,
		DateCompleted:       c.DateCompleted,
		DateExpired:         c.DateExpired,
		DateIssued:          c.DateIssued,
		DaysToComplete:      c.DaysToComplete,
		EndLocationID:       c.EndLocationId,
		ForCorporation:      c.ForCorporation,
		IssuerCorporationID: c.IssuerCorporationId,
		IssuerID:            c.IssuerId,
		Price:               c.Price,
		Reward:              c.Reward,
		StartLocationID:     c.StartLocationId,
		Status:              status,
		Title:               c.Title,
		Type:                typ,
		Volume:              c.Volume,
	}
	id, err := s.st.CreateCharacterContract(ctx, arg)
	if err != nil {
		return err
	}
	items, _, err := s.esiClient.ESI.ContractsApi.GetCharactersCharacterIdContractsContractIdItems(ctx, characterID, c.ContractId, nil)
	if err != nil {
		return err
	}
	typeIDs := set.Of(xslices.Map(items, func(x esi.GetCharactersCharacterIdContractsContractIdItems200Ok) int32 {
		return x.TypeId

	})...)
	if err := s.eus.AddMissingTypes(ctx, typeIDs); err != nil {
		return err
	}
	for _, it := range items {
		arg := storage.CreateCharacterContractItemParams{
			ContractID:  id,
			IsIncluded:  it.IsIncluded,
			IsSingleton: it.IsSingleton,
			Quantity:    it.Quantity,
			RawQuantity: it.RawQuantity,
			RecordID:    it.RecordId,
			TypeID:      it.TypeId,
		}
		if err := s.st.CreateCharacterContractItem(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func (s *CharacterService) updateContract(ctx context.Context, characterID int32, c esi.GetCharactersCharacterIdContracts200Ok) error {
	status, ok := contractStatusFromESIValue[c.Status]
	if !ok {
		return fmt.Errorf("unknown status: %s", c.Status)
	}
	o, err := s.st.GetCharacterContract(ctx, characterID, c.ContractId)
	if err != nil {
		return err
	}
	var acceptorID int32
	if o.Acceptor != nil {
		acceptorID = o.Acceptor.ID
	}
	if c.AcceptorId == acceptorID &&
		c.DateAccepted.Equal(o.DateAccepted.ValueOrZero()) &&
		c.DateCompleted.Equal(o.DateCompleted.ValueOrZero()) &&
		o.Status == contractStatusFromESIValue[c.Status] {
		return nil
	}
	arg := storage.UpdateCharacterContractParams{
		AcceptorID:    c.AcceptorId,
		DateAccepted:  c.DateAccepted,
		DateCompleted: c.DateCompleted,
		CharacterID:   characterID,
		ContractID:    c.ContractId,
		Status:        status,
	}
	if err := s.st.UpdateCharacterContract(ctx, arg); err != nil {
		return err
	}
	return nil
}

func (s *CharacterService) updateContractBids(ctx context.Context, characterID, contractID int32) error {
	c, err := s.st.GetCharacterContract(ctx, characterID, contractID)
	if err != nil {
		return err
	}
	existingBidIDs, err := s.st.ListCharacterContractBidIDs(ctx, c.ID)
	if err != nil {
		return err
	}
	bids, _, err := s.esiClient.ESI.ContractsApi.GetCharactersCharacterIdContractsContractIdBids(ctx, characterID, contractID, nil)
	if err != nil {
		return err
	}
	newBids := make([]esi.GetCharactersCharacterIdContractsContractIdBids200Ok, 0)
	for _, b := range bids {
		if !existingBidIDs.Contains(b.BidId) {
			newBids = append(newBids, b)
		}
	}
	if len(newBids) == 0 {
		return nil
	}
	var eeIDs set.Set[int32]
	for _, b := range newBids {
		if b.BidderId != 0 {
			eeIDs.Add(b.BidderId)
		}
	}
	if eeIDs.Size() > 0 {
		if _, err = s.eus.AddMissingEntities(ctx, eeIDs); err != nil {
			return err
		}
	}
	for _, b := range newBids {
		arg := storage.CreateCharacterContractBidParams{
			ContractID: c.ID,
			Amount:     b.Amount,
			BidID:      b.BidId,
			BidderID:   b.BidderId,
			DateBid:    b.DateBid,
		}
		if err := s.st.CreateCharacterContractBid(ctx, arg); err != nil {
			return err
		}
	}
	slog.Info("created contract bids", "characterID", characterID, "contract", contractID, "count", len(newBids))
	return nil
}

func (s *CharacterService) ListImplants(ctx context.Context, characterID int32) ([]*app.CharacterImplant, error) {
	return s.st.ListCharacterImplants(ctx, characterID)
}

func (s *CharacterService) updateImplantsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionImplants {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			implants, _, err := s.esiClient.ESI.ClonesApi.GetCharactersCharacterIdImplants(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			slog.Debug("Received implants from ESI", "count", len(implants), "characterID", characterID)
			return implants, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			implants := data.([]int32)
			if err := s.eus.AddMissingTypes(ctx, set.Of(implants...)); err != nil {
				return err
			}
			args := make([]storage.CreateCharacterImplantParams, len(implants))
			for i, typeID := range implants {
				args[i] = storage.CreateCharacterImplantParams{
					CharacterID: characterID,
					EveTypeID:   typeID,
				}
			}
			if err := s.st.ReplaceCharacterImplants(ctx, characterID, args); err != nil {
				return err
			}
			slog.Info("Stored updated implants", "characterID", characterID, "count", len(implants))
			return nil
		})
}

// ListAllCharacterIndustryJob returns all industry jobs from characters.
func (s *CharacterService) ListAllCharacterIndustryJob(ctx context.Context) ([]*app.CharacterIndustryJob, error) {
	return s.st.ListAllCharacterIndustryJob(ctx)
}

var jobStatusFromESIValue = map[string]app.IndustryJobStatus{
	"active":    app.JobActive,
	"cancelled": app.JobCancelled,
	"delivered": app.JobDelivered,
	"paused":    app.JobPaused,
	"ready":     app.JobReady,
	"reverted":  app.JobReverted,
}

func (s *CharacterService) updateIndustryJobsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionIndustryJobs {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			jobs, _, err := s.esiClient.ESI.IndustryApi.GetCharactersCharacterIdIndustryJobs(ctx, characterID, &esi.GetCharactersCharacterIdIndustryJobsOpts{
				IncludeCompleted: esioptional.NewBool(true),
			})
			if err != nil {
				return false, err
			}
			slog.Debug("Received industry jobs from ESI", "characterID", characterID, "count", len(jobs))
			return jobs, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			jobs := data.([]esi.GetCharactersCharacterIdIndustryJobs200Ok)
			entityIDs := set.Of[int32]()
			typeIDs := set.Of[int32]()
			locationIDs := set.Of[int64]()
			for _, j := range jobs {
				entityIDs.Add(j.InstallerId)
				if j.CompletedCharacterId != 0 {
					entityIDs.Add(j.CompletedCharacterId)
				}
				locationIDs.Add(j.BlueprintLocationId)
				locationIDs.Add(j.OutputLocationId)
				locationIDs.Add(j.StationId)
				typeIDs.Add(j.BlueprintTypeId)
				if j.ProductTypeId != 0 {
					typeIDs.Add(j.ProductTypeId)
				}
			}
			g := new(errgroup.Group)
			g.Go(func() error {
				_, err := s.eus.AddMissingEntities(ctx, entityIDs)
				return err
			})
			g.Go(func() error {
				return s.eus.AddMissingLocations(ctx, locationIDs)
			})
			g.Go(func() error {
				return s.eus.AddMissingTypes(ctx, typeIDs)
			})
			if err := g.Wait(); err != nil {
				return err
			}
			for _, j := range jobs {
				status, ok := jobStatusFromESIValue[j.Status]
				if !ok {
					status = app.JobUndefined
				}
				if status == app.JobActive && !j.EndDate.IsZero() && j.EndDate.Before(time.Now()) {
					// Workaround for known bug: https://github.com/esi/esi-issues/issues/752
					status = app.JobReady
				}
				arg := storage.UpdateOrCreateCharacterIndustryJobParams{
					ActivityID:           j.ActivityId,
					BlueprintID:          j.BlueprintId,
					BlueprintLocationID:  j.BlueprintLocationId,
					BlueprintTypeID:      j.BlueprintTypeId,
					CharacterID:          characterID,
					CompletedCharacterID: j.CompletedCharacterId,
					CompletedDate:        j.CompletedDate,
					Cost:                 j.Cost,
					Duration:             j.Duration,
					EndDate:              j.EndDate,
					FacilityID:           j.FacilityId,
					InstallerID:          j.InstallerId,
					LicensedRuns:         j.LicensedRuns,
					JobID:                j.JobId,
					OutputLocationID:     j.OutputLocationId,
					Runs:                 j.Runs,
					PauseDate:            j.PauseDate,
					Probability:          j.Probability,
					ProductTypeID:        j.ProductTypeId,
					StartDate:            j.StartDate,
					Status:               status,
					StationID:            j.StationId,
					SuccessfulRuns:       j.SuccessfulRuns,
				}
				if err := s.st.UpdateOrCreateCharacterIndustryJob(ctx, arg); err != nil {
					return nil
				}
			}
			slog.Info("Updated industry jobs", "characterID", characterID, "count", len(jobs))
			return nil
		})
}

func (s *CharacterService) GetJumpClone(ctx context.Context, characterID, cloneID int32) (*app.CharacterJumpClone, error) {
	return s.st.GetCharacterJumpClone(ctx, characterID, cloneID)
}

func (s *CharacterService) ListAllJumpClones(ctx context.Context) ([]*app.CharacterJumpClone2, error) {
	return s.st.ListAllCharacterJumpClones(ctx)
}

func (s *CharacterService) ListJumpClones(ctx context.Context, characterID int32) ([]*app.CharacterJumpClone, error) {
	return s.st.ListCharacterJumpClones(ctx, characterID)
}

// calcNextCloneJump returns when the next clone jump is available.
// It returns a zero time when a jump is available now.
// It returns empty when a jump could not be calculated.
func (s *CharacterService) calcNextCloneJump(ctx context.Context, c *app.Character) (optional.Optional[time.Time], error) {
	var z optional.Optional[time.Time]

	if c.LastCloneJumpAt.IsEmpty() {
		return z, nil
	}
	lastJump := c.LastCloneJumpAt.MustValue()

	var skillLevel int
	sk, err := s.GetSkill(ctx, c.ID, app.EveTypeInfomorphSynchronizing)
	if errors.Is(err, app.ErrNotFound) {
		skillLevel = 0
	} else if err != nil {
		return z, err
	} else {
		skillLevel = sk.ActiveSkillLevel
	}

	nextJump := lastJump.Add(time.Duration(24-skillLevel) * time.Hour)
	if nextJump.Before(time.Now()) {
		return optional.From(time.Time{}), nil
	}
	return optional.From(nextJump), nil
}

// TODO: Consolidate with updating home in separate function

func (s *CharacterService) updateJumpClonesESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionJumpClones {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			clones, _, err := s.esiClient.ESI.ClonesApi.GetCharactersCharacterIdClones(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			slog.Debug("Received jump clones from ESI", "characterID", characterID, "count", len(clones.JumpClones))
			return clones, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			clones := data.(esi.GetCharactersCharacterIdClonesOk)
			var locationIDs set.Set[int64]
			var typeIDs set.Set[int32]
			for _, jc := range clones.JumpClones {
				locationIDs.Add(jc.LocationId)
				typeIDs.AddSeq(slices.Values(jc.Implants))
			}
			if clones.HomeLocation.LocationId != 0 {
				locationIDs.Add(clones.HomeLocation.LocationId)
			}
			g := new(errgroup.Group)
			g.Go(func() error {
				return s.eus.AddMissingLocations(ctx, locationIDs)
			})
			g.Go(func() error {
				return s.eus.AddMissingTypes(ctx, typeIDs)
			})
			if err := g.Wait(); err != nil {
				return err
			}
			args := make([]storage.CreateCharacterJumpCloneParams, len(clones.JumpClones))
			for i, jc := range clones.JumpClones {
				args[i] = storage.CreateCharacterJumpCloneParams{
					CharacterID: characterID,
					Implants:    jc.Implants,
					JumpCloneID: int64(jc.JumpCloneId),
					LocationID:  jc.LocationId,
					Name:        jc.Name,
				}
			}
			if err := s.st.ReplaceCharacterJumpClones(ctx, characterID, args); err != nil {
				return err
			}
			slog.Info("Stored updated jump clones", "characterID", characterID, "count", len(clones.JumpClones))

			var home optional.Optional[int64]
			if clones.HomeLocation.LocationId != 0 {
				home.Set(clones.HomeLocation.LocationId)
			}
			if err := s.st.UpdateCharacterHome(ctx, characterID, home); err != nil {
				return err
			}
			if err := s.st.UpdateCharacterLastCloneJump(ctx, characterID, optional.From(clones.LastCloneJumpDate)); err != nil {
				return err
			}
			return nil
		})
}

// DeleteMail deletes a mail both on ESI and in the database.
func (s *CharacterService) DeleteMail(ctx context.Context, characterID, mailID int32) error {
	token, err := s.GetValidCharacterToken(ctx, characterID)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
	_, err = s.esiClient.ESI.MailApi.DeleteCharactersCharacterIdMailMailId(ctx, characterID, mailID, nil)
	if err != nil {
		return err
	}
	err = s.st.DeleteCharacterMail(ctx, characterID, mailID)
	if err != nil {
		return err
	}
	slog.Info("Mail deleted", "characterID", characterID, "mailID", mailID)
	return nil
}

func (s *CharacterService) GetMail(ctx context.Context, characterID int32, mailID int32) (*app.CharacterMail, error) {
	return s.st.GetCharacterMail(ctx, characterID, mailID)
}

func (s *CharacterService) GetAllMailUnreadCount(ctx context.Context) (int, error) {
	return s.st.GetAllCharactersMailUnreadCount(ctx)
}

// GetMailCounts returns the number of unread mail for a character.
func (s *CharacterService) GetMailCounts(ctx context.Context, characterID int32) (int, int, error) {
	total, err := s.st.GetCharacterMailCount(ctx, characterID)
	if err != nil {
		return 0, 0, err
	}
	unread, err := s.st.GetCharacterMailUnreadCount(ctx, characterID)
	if err != nil {
		return 0, 0, err
	}
	return total, unread, nil
}

func (s *CharacterService) GetMailLabelUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error) {
	return s.st.GetCharacterMailLabelUnreadCounts(ctx, characterID)
}

func (s *CharacterService) GetMailListUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error) {
	return s.st.GetCharacterMailListUnreadCounts(ctx, characterID)
}

func (s *CharacterService) NotifyMails(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error {
	mm, err := s.st.ListCharacterMailHeadersForUnprocessed(ctx, characterID, earliest)
	if err != nil {
		return err
	}
	characterName, err := s.getCharacterName(ctx, characterID)
	if err != nil {
		return err
	}
	for _, m := range mm {
		if m.Timestamp.Before(earliest) {
			continue
		}
		title := fmt.Sprintf("%s: New Mail from %s", characterName, m.From.Name)
		content := m.Subject
		notify(title, content)
		if err := s.st.UpdateCharacterMailSetProcessed(ctx, m.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *CharacterService) ListMailLists(ctx context.Context, characterID int32) ([]*app.EveEntity, error) {
	return s.st.ListCharacterMailListsOrdered(ctx, characterID)
}

// ListMailsForLabel returns a character's mails for a label in descending order by timestamp.
func (s *CharacterService) ListMailHeadersForLabelOrdered(ctx context.Context, characterID int32, labelID int32) ([]*app.CharacterMailHeader, error) {
	return s.st.ListCharacterMailHeadersForLabelOrdered(ctx, characterID, labelID)
}

func (s *CharacterService) ListMailHeadersForListOrdered(ctx context.Context, characterID int32, listID int32) ([]*app.CharacterMailHeader, error) {
	return s.st.ListCharacterMailHeadersForListOrdered(ctx, characterID, listID)
}

func (s *CharacterService) ListMailLabelsOrdered(ctx context.Context, characterID int32) ([]*app.CharacterMailLabel, error) {
	return s.st.ListCharacterMailLabelsOrdered(ctx, characterID)
}

// SendMail creates a new mail on ESI and stores it locally.
func (s *CharacterService) SendMail(ctx context.Context, characterID int32, subject string, recipients []*app.EveEntity, body string) (int32, error) {
	if subject == "" {
		return 0, fmt.Errorf("missing subject")
	}
	if body == "" {
		return 0, fmt.Errorf("missing body")
	}
	if len(recipients) == 0 {
		return 0, fmt.Errorf("missing recipients")
	}
	rr, err := eveEntitiesToESIMailRecipients(recipients)
	if err != nil {
		return 0, err
	}
	token, err := s.GetValidCharacterToken(ctx, characterID)
	if err != nil {
		return 0, err
	}
	esiMail := esi.PostCharactersCharacterIdMailMail{
		Body:       body,
		Subject:    subject,
		Recipients: rr,
	}
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
	mailID, _, err := s.esiClient.ESI.MailApi.PostCharactersCharacterIdMail(ctx, characterID, esiMail, nil)
	if err != nil {
		return 0, err
	}
	recipientIDs := make([]int32, len(rr))
	for i, r := range rr {
		recipientIDs[i] = r.RecipientId
	}
	ids := set.Union(set.Of(recipientIDs...), set.Of(characterID))
	_, err = s.eus.AddMissingEntities(ctx, ids)
	if err != nil {
		return 0, err
	}
	arg1 := storage.MailLabelParams{
		CharacterID: characterID,
		LabelID:     app.MailLabelSent,
		Name:        "Sent",
	}
	_, err = s.st.GetOrCreateCharacterMailLabel(ctx, arg1) // make sure sent label exists
	if err != nil {
		return 0, err
	}
	arg2 := storage.CreateCharacterMailParams{
		Body:         body,
		CharacterID:  characterID,
		FromID:       characterID,
		IsRead:       true,
		LabelIDs:     []int32{app.MailLabelSent},
		MailID:       mailID,
		RecipientIDs: recipientIDs,
		Subject:      subject,
		Timestamp:    time.Now(),
	}
	_, err = s.st.CreateCharacterMail(ctx, arg2)
	if err != nil {
		return 0, err
	}
	slog.Info("Mail sent", "characterID", characterID, "mailID", mailID)
	return mailID, nil
}

var eveEntityCategory2MailRecipientType = map[app.EveEntityCategory]string{
	app.EveEntityAlliance:    "alliance",
	app.EveEntityCharacter:   "character",
	app.EveEntityCorporation: "corporation",
	app.EveEntityMailList:    "mailing_list",
}

func eveEntitiesToESIMailRecipients(ee []*app.EveEntity) ([]esi.PostCharactersCharacterIdMailRecipient, error) {
	rr := make([]esi.PostCharactersCharacterIdMailRecipient, len(ee))
	for i, e := range ee {
		c, ok := eveEntityCategory2MailRecipientType[e.Category]
		if !ok {
			return rr, fmt.Errorf("match EveEntity category to ESI mail recipient type: %v", e)
		}
		rr[i] = esi.PostCharactersCharacterIdMailRecipient{
			RecipientId:   e.ID,
			RecipientType: c,
		}
	}
	return rr, nil
}

const (
	// maxMails              = 1000
	maxMailHeadersPerPage = 50 // maximum header objects returned per page
)

// TODO: Add ability to delete obsolete mail labels

// updateMailLabelsESI updates the skillqueue for a character from ESI
// and reports whether it has changed.
func (s *CharacterService) updateMailLabelsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionMailLabels {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			ll, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailLabels(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			slog.Debug("Received mail labels from ESI", "characterID", characterID, "count", len(ll.Labels))
			return ll, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			ll := data.(esi.GetCharactersCharacterIdMailLabelsOk)
			labels := ll.Labels
			for _, o := range labels {
				arg := storage.MailLabelParams{
					CharacterID: characterID,
					Color:       o.Color,
					LabelID:     o.LabelId,
					Name:        o.Name,
					UnreadCount: int(o.UnreadCount),
				}
				_, err := s.st.UpdateOrCreateCharacterMailLabel(ctx, arg)
				if err != nil {
					return err
				}
			}
			return nil
		})
}

// updateMailListsESI updates the skillqueue for a character from ESI
// and reports whether it has changed.
func (s *CharacterService) updateMailListsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionMailLists {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			lists, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailLists(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return lists, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			lists := data.([]esi.GetCharactersCharacterIdMailLists200Ok)
			for _, o := range lists {
				_, err := s.st.UpdateOrCreateEveEntity(ctx, o.MailingListId, o.Name, app.EveEntityMailList)
				if err != nil {
					return err
				}
				if err := s.st.CreateCharacterMailList(ctx, characterID, o.MailingListId); err != nil {
					return err
				}
			}
			return nil
		})
}

// updateMailsESI updates the skillqueue for a character from ESI
// and reports whether it has changed.
func (s *CharacterService) updateMailsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionMails {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			headers, err := s.fetchMailHeadersESI(ctx, characterID, arg.MaxMails)
			if err != nil {
				return false, err
			}
			slog.Debug("Received mail headers from ESI", "characterID", characterID, "count", len(headers))
			return headers, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			headers := data.([]esi.GetCharactersCharacterIdMail200Ok)
			newHeaders, existingHeaders, err := s.determineNewMail(ctx, characterID, headers)
			if err != nil {
				return err
			}
			if len(newHeaders) > 0 {
				if err := s.resolveMailEntities(ctx, newHeaders); err != nil {
					return err
				}
				if err := s.addNewMailsESI(ctx, characterID, newHeaders); err != nil {
					return err
				}
			}
			if len(existingHeaders) > 0 {
				if err := s.updateExistingMail(ctx, characterID, existingHeaders); err != nil {
					return err
				}
			}
			// TODO: Delete obsolete mail labels and list
			// if err := s.st.DeleteObsoleteCharacterMailLabels(ctx, characterID); err != nil {
			// 	return err
			// }
			// if err := s.st.DeleteObsoleteCharacterMailLists(ctx, characterID); err != nil {
			// 	return err
			// }
			return nil
		})
}

// fetchMailHeadersESI fetched mail headers from ESI with paging and returns them.
func (s *CharacterService) fetchMailHeadersESI(ctx context.Context, characterID int32, maxMails int) ([]esi.GetCharactersCharacterIdMail200Ok, error) {
	var oo2 []esi.GetCharactersCharacterIdMail200Ok
	lastMailID := int32(0)
	for {
		var opts *esi.GetCharactersCharacterIdMailOpts
		if lastMailID > 0 {
			opts = &esi.GetCharactersCharacterIdMailOpts{LastMailId: esioptional.NewInt32(lastMailID)}
		} else {
			opts = nil
		}
		oo, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMail(ctx, characterID, opts)
		if err != nil {
			return nil, err
		}
		oo2 = slices.Concat(oo2, oo)
		isLimitExceeded := (maxMails != 0 && len(oo2)+maxMailHeadersPerPage > maxMails)
		if len(oo) < maxMailHeadersPerPage || isLimitExceeded {
			break
		}
		ids := make([]int32, len(oo))
		for i, o := range oo {
			ids[i] = o.MailId
		}
		lastMailID = slices.Min(ids)
	}
	slog.Debug("Received mail headers", "characterID", characterID, "count", len(oo2))
	return oo2, nil
}

func (s *CharacterService) determineNewMail(ctx context.Context, characterID int32, mm []esi.GetCharactersCharacterIdMail200Ok) ([]esi.GetCharactersCharacterIdMail200Ok, []esi.GetCharactersCharacterIdMail200Ok, error) {
	newMail := make([]esi.GetCharactersCharacterIdMail200Ok, 0, len(mm))
	existingMail := make([]esi.GetCharactersCharacterIdMail200Ok, 0, len(mm))
	existingIDs, _, err := s.determineMailIDs(ctx, characterID, mm)
	if err != nil {
		return newMail, existingMail, err
	}
	for _, h := range mm {
		if existingIDs.Contains(h.MailId) {
			existingMail = append(existingMail, h)
		} else {
			newMail = append(newMail, h)
		}
	}
	return newMail, existingMail, nil
}

func (s *CharacterService) determineMailIDs(ctx context.Context, characterID int32, headers []esi.GetCharactersCharacterIdMail200Ok) (set.Set[int32], set.Set[int32], error) {
	existingIDs, err := s.st.ListCharacterMailIDs(ctx, characterID)
	if err != nil {
		return set.Of[int32](), set.Of[int32](), err
	}
	incomingIDs := set.Of[int32]()
	for _, h := range headers {
		incomingIDs.Add(h.MailId)
	}
	missingIDs := set.Difference(incomingIDs, existingIDs)
	return existingIDs, missingIDs, nil
}

func (s *CharacterService) resolveMailEntities(ctx context.Context, mm []esi.GetCharactersCharacterIdMail200Ok) error {
	entityIDs := set.Of[int32]()
	for _, m := range mm {
		entityIDs.Add(m.From)
		for _, r := range m.Recipients {
			entityIDs.Add(r.RecipientId)
		}
	}
	_, err := s.eus.AddMissingEntities(ctx, entityIDs)
	if err != nil {
		return err
	}
	return nil
}

func (s *CharacterService) addNewMailsESI(ctx context.Context, characterID int32, headers []esi.GetCharactersCharacterIdMail200Ok) error {
	type esiMailWrapper struct {
		mail esi.GetCharactersCharacterIdMailMailIdOk
		id   int32
	}
	mails := make([]esiMailWrapper, len(headers))
	g := new(errgroup.Group)
	for i, h := range headers {
		g.Go(func() error {
			m, _, err := s.esiClient.ESI.MailApi.GetCharactersCharacterIdMailMailId(ctx, characterID, h.MailId, nil)
			if err != nil {
				return err
			}
			mails[i].mail = m
			mails[i].id = h.MailId
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	for _, m := range mails {
		recipientIDs := make([]int32, len(m.mail.Recipients))
		for i, r := range m.mail.Recipients {
			recipientIDs[i] = r.RecipientId
		}
		arg := storage.CreateCharacterMailParams{
			Body:         m.mail.Body,
			CharacterID:  characterID,
			FromID:       m.mail.From,
			IsRead:       m.mail.Read,
			LabelIDs:     m.mail.Labels,
			MailID:       m.id,
			RecipientIDs: recipientIDs,
			Subject:      m.mail.Subject,
			Timestamp:    m.mail.Timestamp,
		}
		_, err := s.st.CreateCharacterMail(ctx, arg)
		if err != nil {
			return err
		}
	}
	slog.Info("Stored new mail", "characterID", characterID, "count", len(mails))
	return nil
}

func (s *CharacterService) updateExistingMail(ctx context.Context, characterID int32, headers []esi.GetCharactersCharacterIdMail200Ok) error {
	var updated int
	for _, h := range headers {
		m, err := s.st.GetCharacterMail(ctx, characterID, h.MailId)
		if err != nil {
			return err
		}
		if m.IsRead != h.IsRead {
			err := s.st.UpdateCharacterMail(ctx, characterID, m.ID, h.IsRead, h.Labels)
			if err != nil {
				return err
			}
			updated++
		}
	}
	if updated > 0 {
		slog.Info("Updated mail", "characterID", characterID, "count", updated)
	}
	return nil
}

// UpdateMailRead updates an existing mail as read
func (s *CharacterService) UpdateMailRead(ctx context.Context, characterID, mailID int32) error {
	token, err := s.GetValidCharacterToken(ctx, characterID)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
	m, err := s.st.GetCharacterMail(ctx, characterID, mailID)
	if err != nil {
		return err
	}
	labelIDs := make([]int32, len(m.Labels))
	for i, l := range m.Labels {
		labelIDs[i] = l.LabelID
	}
	contents := esi.PutCharactersCharacterIdMailMailIdContents{Read: true, Labels: labelIDs}
	_, err = s.esiClient.ESI.MailApi.PutCharactersCharacterIdMailMailId(ctx, m.CharacterID, contents, m.MailID, nil)
	if err != nil {
		return err
	}
	m.IsRead = true
	if err := s.st.UpdateCharacterMail(ctx, characterID, m.ID, m.IsRead, labelIDs); err != nil {
		return err
	}
	return nil

}

func (s *CharacterService) CountNotifications(ctx context.Context, characterID int32) (map[app.NotificationGroup][]int, error) {
	types, err := s.st.CountCharacterNotifications(ctx, characterID)
	if err != nil {
		return nil, err
	}
	values := make(map[app.NotificationGroup][]int)
	for name, v := range types {
		c := evenotification.Type2group[evenotification.Type(name)]
		if _, ok := values[c]; !ok {
			values[c] = make([]int, 2)
		}
		values[c][0] += v[0]
		values[c][1] += v[1]
	}
	return values, nil
}

// TODO: Add tests for NotifyCommunications

func (s *CharacterService) NotifyCommunications(ctx context.Context, characterID int32, earliest time.Time, typesEnabled set.Set[string], notify func(title, content string)) error {
	nn, err := s.st.ListCharacterNotificationsUnprocessed(ctx, characterID, earliest)
	if err != nil {
		return err
	}
	if len(nn) == 0 {
		return nil
	}
	characterName, err := s.getCharacterName(ctx, characterID)
	if err != nil {
		return err
	}
	for _, n := range nn {
		if !typesEnabled.Contains(n.Type) {
			continue
		}
		title := fmt.Sprintf("%s: New Communication from %s", characterName, n.Sender.Name)
		content := n.Title.ValueOrZero()
		notify(title, content)
		if err := s.st.UpdateCharacterNotificationSetProcessed(ctx, n.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *CharacterService) ListNotificationsTypes(ctx context.Context, characterID int32, ng app.NotificationGroup) ([]*app.CharacterNotification, error) {
	types := evenotification.GroupTypes[ng]
	t2 := make([]string, len(types))
	for i, v := range types {
		t2[i] = string(v)
	}
	return s.st.ListCharacterNotificationsTypes(ctx, characterID, t2)
}

func (s *CharacterService) ListNotificationsAll(ctx context.Context, characterID int32) ([]*app.CharacterNotification, error) {
	return s.st.ListCharacterNotificationsAll(ctx, characterID)
}

func (s *CharacterService) ListNotificationsUnread(ctx context.Context, characterID int32) ([]*app.CharacterNotification, error) {
	return s.st.ListCharacterNotificationsUnread(ctx, characterID)
}

func (s *CharacterService) updateNotificationsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionNotifications {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			notifications, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterIdNotifications(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			slog.Debug("Received notifications from ESI", "characterID", characterID, "count", len(notifications))
			return notifications, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			notifications := data.([]esi.GetCharactersCharacterIdNotifications200Ok)
			existingIDs, err := s.st.ListCharacterNotificationIDs(ctx, characterID)
			if err != nil {
				return err
			}
			var newNotifs []esi.GetCharactersCharacterIdNotifications200Ok
			var existingNotifs []esi.GetCharactersCharacterIdNotifications200Ok
			for _, n := range notifications {
				if existingIDs.Contains(n.NotificationId) {
					existingNotifs = append(existingNotifs, n)
				} else {
					newNotifs = append(newNotifs, n)
				}
			}
			if err := s.loadEntitiesForNotifications(ctx, existingNotifs); err != nil {
				return err
			}
			var updatedCount int
			for _, n := range existingNotifs {
				o, err := s.st.GetCharacterNotification(ctx, characterID, n.NotificationId)
				if err != nil {
					slog.Error("Failed to get existing character notification", "characterID", characterID, "NotificationID", n.NotificationId, "error", err)
					continue
				}
				arg1 := storage.UpdateCharacterNotificationParams{
					ID:     o.ID,
					IsRead: o.IsRead,
					Title:  o.Title,
					Body:   o.Body,
				}
				arg2 := storage.UpdateCharacterNotificationParams{
					ID:     o.ID,
					IsRead: n.IsRead,
				}
				title, body, err := s.ens.RenderESI(ctx, n.Type_, n.Text, n.Timestamp)
				if errors.Is(err, app.ErrNotFound) {
					// do nothing
				} else if err != nil {
					slog.Error("Failed to render character notification", "characterID", characterID, "NotificationID", n.NotificationId, "error", err)
				} else {
					arg2.Title.Set(title)
					arg2.Body.Set(body)
				}
				if arg2 != arg1 {
					if err := s.st.UpdateCharacterNotification(ctx, arg2); err != nil {
						return err
					}
					updatedCount++
				}
			}
			if updatedCount > 0 {
				slog.Info("Updated notifications", "characterID", characterID, "count", updatedCount)
			}
			if len(newNotifs) == 0 {
				slog.Info("No new notifications", "characterID", characterID)
				return nil
			}
			if err := s.loadEntitiesForNotifications(ctx, newNotifs); err != nil {
				return err
			}
			args := make([]storage.CreateCharacterNotificationParams, len(newNotifs))
			g := new(errgroup.Group)
			for i, n := range newNotifs {
				g.Go(func() error {
					arg := storage.CreateCharacterNotificationParams{
						CharacterID:    characterID,
						IsRead:         n.IsRead,
						NotificationID: n.NotificationId,
						SenderID:       n.SenderId,
						Text:           n.Text,
						Timestamp:      n.Timestamp,
						Type:           n.Type_,
					}
					title, body, err := s.ens.RenderESI(ctx, n.Type_, n.Text, n.Timestamp)
					if errors.Is(err, app.ErrNotFound) {
						// do nothing
					} else if err != nil {
						slog.Error("Failed to render character notification", "characterID", characterID, "NotificationID", n.NotificationId, "error", err)
					} else {
						arg.Title.Set(title)
						arg.Body.Set(body)
					}
					args[i] = arg
					return nil
				})
			}
			if err := g.Wait(); err != nil {
				return err
			}
			for _, arg := range args {
				if err := s.st.CreateCharacterNotification(ctx, arg); err != nil {
					return err
				}
			}
			slog.Info("Stored new notifications", "characterID", characterID, "entries", len(newNotifs))
			return nil
		})
}

func (s *CharacterService) loadEntitiesForNotifications(ctx context.Context, notifications []esi.GetCharactersCharacterIdNotifications200Ok) error {
	if len(notifications) == 0 {
		return nil
	}
	var ids set.Set[int32]
	for _, n := range notifications {
		if n.SenderId != 0 {
			ids.Add(n.SenderId)
		}
		ids2, err := s.ens.EntityIDs(n.Type_, n.Text)
		if errors.Is(err, app.ErrNotFound) {
			continue
		} else if err != nil {
			return err
		}
		ids.AddSeq(ids2.All())
	}
	if ids.Size() > 0 {
		_, err := s.eus.AddMissingEntities(ctx, ids)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *CharacterService) NotifyExpiredExtractions(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error {
	planets, err := s.ListPlanets(ctx, characterID)
	if err != nil {
		return err
	}
	characterName, err := s.getCharacterName(ctx, characterID)
	if err != nil {
		return err
	}
	for _, p := range planets {
		expiration := p.ExtractionsExpiryTime()
		if expiration.IsZero() || expiration.After(time.Now()) || expiration.Before(earliest) {
			continue
		}
		if p.LastNotified.ValueOrZero().Equal(expiration) {
			continue
		}
		title := fmt.Sprintf("%s: PI extraction expired", characterName)
		extracted := strings.Join(p.ExtractedTypeNames(), ",")
		content := fmt.Sprintf("Extraction expired at %s for %s", p.EvePlanet.Name, extracted)
		notify(title, content)
		arg := storage.UpdateCharacterPlanetLastNotifiedParams{
			CharacterID:  characterID,
			EvePlanetID:  p.EvePlanet.ID,
			LastNotified: expiration,
		}
		if err := s.st.UpdateCharacterPlanetLastNotified(ctx, arg); err != nil {
			return err
		}
	}
	return nil
}

func (s *CharacterService) GetPlanet(ctx context.Context, characterID, planetID int32) (*app.CharacterPlanet, error) {
	return s.st.GetCharacterPlanet(ctx, characterID, planetID)
}

func (s *CharacterService) ListAllPlanets(ctx context.Context) ([]*app.CharacterPlanet, error) {
	return s.st.ListAllCharacterPlanets(ctx)
}

func (s *CharacterService) ListPlanets(ctx context.Context, characterID int32) ([]*app.CharacterPlanet, error) {
	return s.st.ListCharacterPlanets(ctx, characterID)
}

// TODO: Improve update logic to only update changes

func (s *CharacterService) updatePlanetsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionPlanets {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			planets, _, err := s.esiClient.ESI.PlanetaryInteractionApi.GetCharactersCharacterIdPlanets(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			slog.Debug("Received planets from ESI", "characterID", characterID, "count", len(planets))
			return planets, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			// remove obsolete planets
			pp, err := s.st.ListCharacterPlanets(ctx, characterID)
			if err != nil {
				return err
			}
			existing := set.Of[int32]()
			for _, p := range pp {
				existing.Add(p.EvePlanet.ID)
			}
			planets := data.([]esi.GetCharactersCharacterIdPlanets200Ok)
			incoming := set.Of[int32]()
			for _, p := range planets {
				incoming.Add(p.PlanetId)
			}
			obsolete := set.Difference(existing, incoming)
			if err := s.st.DeleteCharacterPlanet(ctx, characterID, obsolete.Slice()); err != nil {
				return err
			}
			// update or create planet
			for _, o := range planets {
				_, err := s.eus.GetOrCreatePlanetESI(ctx, o.PlanetId)
				if err != nil {
					return err
				}
				arg := storage.UpdateOrCreateCharacterPlanetParams{
					CharacterID:  characterID,
					EvePlanetID:  o.PlanetId,
					LastUpdate:   o.LastUpdate,
					UpgradeLevel: int(o.UpgradeLevel),
				}
				characterPlanetID, err := s.st.UpdateOrCreateCharacterPlanet(ctx, arg)
				if err != nil {
					return err
				}
				planet, _, err := s.esiClient.ESI.PlanetaryInteractionApi.GetCharactersCharacterIdPlanetsPlanetId(ctx, characterID, o.PlanetId, nil)
				if err != nil {
					return err
				}
				// replace planet pins
				if err := s.st.DeletePlanetPins(ctx, characterPlanetID); err != nil {
					return err
				}
				for _, pin := range planet.Pins {
					et, err := s.eus.GetOrCreateTypeESI(ctx, pin.TypeId)
					if err != nil {
						return err
					}
					arg := storage.CreatePlanetPinParams{
						CharacterPlanetID: characterPlanetID,
						TypeID:            et.ID,
						PinID:             pin.PinId,
						ExpiryTime:        pin.ExpiryTime,
						InstallTime:       pin.InstallTime,
						LastCycleStart:    pin.LastCycleStart,
					}
					if pin.ExtractorDetails.ProductTypeId != 0 {
						et, err := s.eus.GetOrCreateTypeESI(ctx, pin.ExtractorDetails.ProductTypeId)
						if err != nil {
							return err
						}
						arg.ExtractorProductTypeID = optional.From(et.ID)
					}
					if pin.FactoryDetails.SchematicId != 0 {
						es, err := s.eus.GetOrCreateSchematicESI(ctx, pin.FactoryDetails.SchematicId)
						if err != nil {
							return err
						}
						arg.FactorySchemaID = optional.From(es.ID)
					}
					if pin.SchematicId != 0 {
						es, err := s.eus.GetOrCreateSchematicESI(ctx, pin.SchematicId)
						if err != nil {
							return err
						}
						arg.SchematicID = optional.From(es.ID)
					}
					if err := s.st.CreatePlanetPin(ctx, arg); err != nil {
						return err
					}
				}
			}
			slog.Info("Stored updated planets", "characterID", characterID, "count", len(planets))
			return nil
		})
}

func (s *CharacterService) ListRoles(ctx context.Context, characterID int32) ([]app.CharacterRole, error) {
	granted, err := s.st.ListCharacterRoles(ctx, characterID)
	if err != nil {
		return nil, err
	}
	rolesSorted := slices.SortedFunc(app.CorporationRoles(), func(a, b app.Role) int {
		return strings.Compare(a.String(), b.String())
	})
	roles := make([]app.CharacterRole, 0)
	for _, r := range rolesSorted {
		o := app.CharacterRole{
			CharacterID: characterID,
			Role:        r,
			Granted:     granted.Contains(r),
		}
		roles = append(roles, o)
	}
	return roles, nil
}

// Roles
func (s *CharacterService) updateRolesESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionRoles {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	roleMap := map[string]app.Role{
		"Account_Take_1":            app.RoleAccountTake1,
		"Account_Take_2":            app.RoleAccountTake2,
		"Account_Take_3":            app.RoleAccountTake3,
		"Account_Take_4":            app.RoleAccountTake4,
		"Account_Take_5":            app.RoleAccountTake5,
		"Account_Take_6":            app.RoleAccountTake6,
		"Account_Take_7":            app.RoleAccountTake7,
		"Accountant":                app.RoleAccountant,
		"Auditor":                   app.RoleAuditor,
		"Brand_Manager":             app.RoleBrandManager,
		"Communications_Officer":    app.RoleCommunicationsOfficer,
		"Config_Equipment":          app.RoleConfigEquipment,
		"Config_Starbase_Equipment": app.RoleConfigStarbaseEquipment,
		"Container_Take_1":          app.RoleContainerTake1,
		"Container_Take_2":          app.RoleContainerTake2,
		"Container_Take_3":          app.RoleContainerTake3,
		"Container_Take_4":          app.RoleContainerTake4,
		"Container_Take_5":          app.RoleContainerTake5,
		"Container_Take_6":          app.RoleContainerTake6,
		"Container_Take_7":          app.RoleContainerTake7,
		"Contract_Manager":          app.RoleContractManager,
		"Deliveries_Container_Take": app.RoleDeliveriesContainerTake,
		"Deliveries_Query":          app.RoleDeliveriesQuery,
		"Deliveries_Take":           app.RoleDeliveriesTake,
		"Diplomat":                  app.RoleDiplomat,
		"Director":                  app.RoleDirector,
		"Factory_Manager":           app.RoleFactoryManager,
		"Fitting_Manager":           app.RoleFittingManager,
		"Hangar_Query_1":            app.RoleHangarQuery1,
		"Hangar_Query_2":            app.RoleHangarQuery2,
		"Hangar_Query_3":            app.RoleHangarQuery3,
		"Hangar_Query_4":            app.RoleHangarQuery4,
		"Hangar_Query_5":            app.RoleHangarQuery5,
		"Hangar_Query_6":            app.RoleHangarQuery6,
		"Hangar_Query_7":            app.RoleHangarQuery7,
		"Hangar_Take_1":             app.RoleHangarTake1,
		"Hangar_Take_2":             app.RoleHangarTake2,
		"Hangar_Take_3":             app.RoleHangarTake3,
		"Hangar_Take_4":             app.RoleHangarTake4,
		"Hangar_Take_5":             app.RoleHangarTake5,
		"Hangar_Take_6":             app.RoleHangarTake6,
		"Hangar_Take_7":             app.RoleHangarTake7,
		"Junior_Accountant":         app.RoleJuniorAccountant,
		"Personnel_Manager":         app.RolePersonnelManager,
		"Project_Manager":           app.RoleProjectManager,
		"Rent_Factory_Facility":     app.RoleRentFactoryFacility,
		"Rent_Office":               app.RoleRentOffice,
		"Rent_Research_Facility":    app.RoleRentResearchFacility,
		"Security_Officer":          app.RoleSecurityOfficer,
		"Skill_Plan_Manager":        app.RoleSkillPlanManager,
		"Starbase_Defense_Operator": app.RoleStarbaseDefenseOperator,
		"Starbase_Fuel_Technician":  app.RoleStarbaseFuelTechnician,
		"Station_Manager":           app.RoleStationManager,
		"Trader":                    app.RoleTrader,
	}

	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			roles, _, err := s.esiClient.ESI.CharacterApi.GetCharactersCharacterIdRoles(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			return roles, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			r := data.(esi.GetCharactersCharacterIdRolesOk)
			var roles set.Set[app.Role]
			for _, n := range r.Roles {
				r, ok := roleMap[n]
				if !ok {
					slog.Warn("received unknown role from ESI", "characterID", characterID, "role", n)
				}
				roles.Add(r)
			}
			return s.st.UpdateCharacterRoles(ctx, characterID, roles)
		})
}

// SearchESI performs a name search for items on the ESI server
// and returns the results by EveEntity category and sorted by name.
// It also returns the total number of results.
// A total of 500 indicates that we exceeded the server limit.
func (s *CharacterService) SearchESI(
	ctx context.Context,
	characterID int32,
	search string,
	categories []app.SearchCategory, strict bool,
) (map[app.SearchCategory][]*app.EveEntity, int, error) {
	token, err := s.GetValidCharacterToken(ctx, characterID)
	if err != nil {
		return nil, 0, err
	}
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
	cc := xslices.Map(categories, func(a app.SearchCategory) string {
		return string(a)
	})
	x, _, err := s.esiClient.ESI.SearchApi.GetCharactersCharacterIdSearch(
		ctx,
		cc,
		characterID,
		search,
		&esi.GetCharactersCharacterIdSearchOpts{
			Strict: esioptional.NewBool(strict),
		})
	if err != nil {
		return nil, 0, err
	}
	ids := set.Of(slices.Concat(
		x.Agent,
		x.Alliance,
		x.Character,
		x.Corporation,
		x.Constellation,
		x.Faction,
		x.InventoryType,
		x.SolarSystem,
		x.Station,
		x.Region,
	)...)
	eeMap, err := s.eus.ToEntities(ctx, ids)
	if err != nil {
		slog.Error("SearchESI: resolve IDs to eve entities", "error", err)
		return nil, 0, err
	}
	categoryMap := map[app.SearchCategory][]int32{
		app.SearchAgent:         x.Agent,
		app.SearchAlliance:      x.Alliance,
		app.SearchCharacter:     x.Character,
		app.SearchConstellation: x.Constellation,
		app.SearchCorporation:   x.Corporation,
		app.SearchFaction:       x.Faction,
		app.SearchRegion:        x.Region,
		app.SearchSolarSystem:   x.SolarSystem,
		app.SearchStation:       x.Station,
		app.SearchType:          x.InventoryType,
	}
	r := make(map[app.SearchCategory][]*app.EveEntity)
	for c, ids2 := range categoryMap {
		for _, id := range ids2 {
			r[c] = append(r[c], eeMap[id])
		}
	}
	for _, s := range r {
		slices.SortFunc(s, func(a, b *app.EveEntity) int {
			return a.Compare(b)
		})
	}
	return r, ids.Size(), nil
}

func (s *CharacterService) ListShipsAbilities(ctx context.Context, characterID int32, search string) ([]*app.CharacterShipAbility, error) {
	return s.st.ListCharacterShipsAbilities(ctx, characterID, search)
}

func (s *CharacterService) GetSkill(ctx context.Context, characterID, typeID int32) (*app.CharacterSkill, error) {
	return s.st.GetCharacterSkill(ctx, characterID, typeID)
}

func (s *CharacterService) ListAllCharactersIndustrySlots(ctx context.Context, typ app.IndustryJobType) ([]app.CharacterIndustrySlots, error) {
	total := make(map[int32]int)
	switch typ {
	case app.ManufacturingJob:
		industry1, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeIndustry)
		if err != nil {
			return nil, err
		}
		for _, r := range industry1 {
			if r.Level > 0 {
				total[r.CharacterID] += 1
			}
		}
		industry2, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeMassProduction)
		if err != nil {
			return nil, err
		}
		for _, r := range industry2 {
			total[r.CharacterID] += r.Level
		}
		industry3, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeAdvancedMassProduction)
		if err != nil {
			return nil, err
		}
		for _, r := range industry3 {
			total[r.CharacterID] += r.Level
		}
	case app.ScienceJob:
		research1, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeLaboratoryOperation)
		if err != nil {
			return nil, err
		}
		for _, r := range research1 {
			total[r.CharacterID] += r.Level + 1 // also adds base slot
		}
		research2, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeAdvancedLaboratoryOperation)
		if err != nil {
			return nil, err
		}
		for _, r := range research2 {
			total[r.CharacterID] += r.Level
		}
	case app.ReactionJob:
		reactions1, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeMassReactions)
		if err != nil {
			return nil, err
		}
		for _, r := range reactions1 {
			total[r.CharacterID] += r.Level + 1 // also adds base slot
		}
		reactions2, err := s.st.ListAllCharactersActiveSkillLevels(ctx, app.EveTypeAdvancedMassReactions)
		if err != nil {
			return nil, err
		}
		for _, r := range reactions2 {
			total[r.CharacterID] += r.Level
		}
	}
	characters, err := s.st.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	results := make(map[int32]app.CharacterIndustrySlots)
	for _, c := range characters {
		results[c.ID] = app.CharacterIndustrySlots{
			CharacterID:   c.ID,
			CharacterName: c.Name,
			Type:          typ,
			Total:         total[c.ID],
		}
	}
	counts, err := s.st.ListAllCharacterIndustryJobActiveCounts(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range counts {
		if !typ.Activities().Contains(r.Activity) {
			continue
		}
		x := results[r.InstallerID]
		switch r.Status {
		case app.JobActive:
			x.Busy += r.Count
		case app.JobReady:
			x.Ready += r.Count
		}
		results[r.InstallerID] = x
	}
	for id, r := range results {
		r.Free = r.Total - r.Busy - r.Ready
		results[id] = r
	}
	rows := slices.SortedFunc(maps.Values(results), func(a, b app.CharacterIndustrySlots) int {
		return strings.Compare(a.CharacterName, b.CharacterName)
	})
	return rows, nil
}

func (s *CharacterService) ListSkillProgress(ctx context.Context, characterID, eveGroupID int32) ([]app.ListSkillProgress, error) {
	return s.st.ListCharacterSkillProgress(ctx, characterID, eveGroupID)
}

func (s *CharacterService) ListSkillGroupsProgress(ctx context.Context, characterID int32) ([]app.ListCharacterSkillGroupProgress, error) {
	return s.st.ListCharacterSkillGroupsProgress(ctx, characterID)
}

func (s *CharacterService) updateSkillsESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionSkills {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			skills, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkills(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			slog.Debug("Received character skills from ESI", "characterID", characterID, "items", len(skills.Skills))
			return skills, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			skills := data.(esi.GetCharactersCharacterIdSkillsOk)
			total := optional.From(int(skills.TotalSp))
			unallocated := optional.From(int(skills.UnallocatedSp))
			if err := s.st.UpdateCharacterSkillPoints(ctx, characterID, total, unallocated); err != nil {
				return err
			}
			currentSkillIDs, err := s.st.ListCharacterSkillIDs(ctx, characterID)
			if err != nil {
				return err
			}
			incomingSkillIDs := set.Of[int32]()
			for _, o := range skills.Skills {
				incomingSkillIDs.Add(o.SkillId)
				_, err := s.eus.GetOrCreateTypeESI(ctx, o.SkillId)
				if err != nil {
					return err
				}
				arg := storage.UpdateOrCreateCharacterSkillParams{
					CharacterID:        characterID,
					EveTypeID:          o.SkillId,
					ActiveSkillLevel:   int(o.ActiveSkillLevel),
					TrainedSkillLevel:  int(o.TrainedSkillLevel),
					SkillPointsInSkill: int(o.SkillpointsInSkill),
				}
				err = s.st.UpdateOrCreateCharacterSkill(ctx, arg)
				if err != nil {
					return err
				}
			}
			slog.Info("Stored updated character skills", "characterID", characterID, "count", len(skills.Skills))
			if ids := set.Difference(currentSkillIDs, incomingSkillIDs); ids.Size() > 0 {
				if err := s.st.DeleteCharacterSkills(ctx, characterID, ids.Slice()); err != nil {
					return err
				}
				slog.Info("Deleted obsolete character skills", "characterID", characterID, "count", ids.Size())
			}
			return nil
		})
}

func (s *CharacterService) GetTotalTrainingTime(ctx context.Context, characterID int32) (optional.Optional[time.Duration], error) {
	return s.st.GetCharacterTotalTrainingTime(ctx, characterID)
}

func (s *CharacterService) NotifyExpiredTraining(ctx context.Context, characterID int32, notify func(title, content string)) error {
	c, err := s.GetCharacter(ctx, characterID)
	if err != nil {
		return err
	}
	if !c.IsTrainingWatched {
		return nil
	}
	t, err := s.GetTotalTrainingTime(ctx, characterID)
	if err != nil {
		return err
	}
	if !t.IsEmpty() {
		return nil
	}
	title := fmt.Sprintf("%s: No skill in training", c.EveCharacter.Name)
	content := "There is currently no skill being trained for this character."
	notify(title, content)
	return s.UpdateIsTrainingWatched(ctx, characterID, false)
}

func (s *CharacterService) ListSkillqueueItems(ctx context.Context, characterID int32) ([]*app.CharacterSkillqueueItem, error) {
	return s.st.ListCharacterSkillqueueItems(ctx, characterID)
}

// UpdateSkillqueueESI updates the skillqueue for a character from ESI
// and reports whether it has changed.
func (s *CharacterService) UpdateSkillqueueESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionSkillqueue {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			items, _, err := s.esiClient.ESI.SkillsApi.GetCharactersCharacterIdSkillqueue(ctx, characterID, nil)
			if err != nil {
				return false, err
			}
			slog.Debug("Received skillqueue from ESI", "characterID", characterID, "items", len(items))
			return items, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			items := data.([]esi.GetCharactersCharacterIdSkillqueue200Ok)
			args := make([]storage.SkillqueueItemParams, len(items))
			for i, o := range items {
				_, err := s.eus.GetOrCreateTypeESI(ctx, o.SkillId)
				if err != nil {
					return err
				}
				args[i] = storage.SkillqueueItemParams{
					EveTypeID:       o.SkillId,
					FinishDate:      o.FinishDate,
					FinishedLevel:   int(o.FinishedLevel),
					LevelEndSP:      int(o.LevelEndSp),
					LevelStartSP:    int(o.LevelStartSp),
					CharacterID:     characterID,
					QueuePosition:   int(o.QueuePosition),
					StartDate:       o.StartDate,
					TrainingStartSP: int(o.TrainingStartSp),
				}
			}
			if err := s.st.ReplaceCharacterSkillqueueItems(ctx, characterID, args); err != nil {
				return err
			}
			slog.Info("Stored updated skillqueue items", "characterID", characterID, "count", len(args))
			return nil
		})

}

// HasTokenWithScopes reports whether a character's token has the requested scopes.
func (s *CharacterService) HasTokenWithScopes(ctx context.Context, characterID int32) (bool, error) {
	t, err := s.st.GetCharacterToken(ctx, characterID)
	if errors.Is(err, app.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	current := set.Of(t.Scopes...)
	required := set.Of(esiScopes...)
	hasScope := current.ContainsAll(required.All())
	return hasScope, nil
}

func (s *CharacterService) ValidCharacterTokenForCorporation(ctx context.Context, corporationID int32, role app.Role) (*app.CharacterToken, error) {
	token, err := s.st.ListCharacterTokenForCorporation(ctx, corporationID, role)
	if err != nil {
		return nil, err
	}
	for _, t := range token {
		err := s.ensureValidCharacterToken(ctx, t)
		if err != nil {
			slog.Error("Failed to refresh token for corporation", "characterID", t.CharacterID, "corporationID", corporationID, "role", role)
			continue
		}
		return t, nil
	}
	return nil, app.ErrNotFound
}

// GetValidCharacterToken returns a valid token for a character.
// Will automatically try to refresh a token if needed.
func (s *CharacterService) GetValidCharacterToken(ctx context.Context, characterID int32) (*app.CharacterToken, error) {
	t, err := s.st.GetCharacterToken(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureValidCharacterToken(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// ensureValidCharacterToken will automatically try to refresh a token that is already or about to become invalid.
func (s *CharacterService) ensureValidCharacterToken(ctx context.Context, t *app.CharacterToken) error {
	if t.RemainsValid(time.Second * 60) {
		return nil
	}
	slog.Debug("Need to refresh token", "characterID", t.CharacterID)
	rawToken, err := s.sso.RefreshToken(ctx, t.RefreshToken)
	if err != nil {
		return err
	}
	t.AccessToken = rawToken.AccessToken
	t.RefreshToken = rawToken.RefreshToken
	t.ExpiresAt = rawToken.ExpiresAt
	err = s.st.UpdateOrCreateCharacterToken(ctx, t)
	if err != nil {
		return err
	}
	slog.Info("Token refreshed", "characterID", t.CharacterID)
	return nil
}

func (s *CharacterService) ListWalletJournalEntries(ctx context.Context, characterID int32) ([]*app.CharacterWalletJournalEntry, error) {
	return s.st.ListCharacterWalletJournalEntries(ctx, characterID)
}

// updateWalletJournalEntryESI updates the wallet journal from ESI and reports whether it has changed.
func (s *CharacterService) updateWalletJournalEntryESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionWalletJournal {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			entries, err := fetchFromESIWithPaging(
				func(pageNum int) ([]esi.GetCharactersCharacterIdWalletJournal200Ok, *http.Response, error) {
					arg := &esi.GetCharactersCharacterIdWalletJournalOpts{
						Page: esioptional.NewInt32(int32(pageNum)),
					}
					return s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWalletJournal(ctx, characterID, arg)
				})
			if err != nil {
				return false, err
			}
			slog.Debug("Received wallet journal from ESI", "entries", len(entries), "characterID", characterID)
			return entries, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			entries := data.([]esi.GetCharactersCharacterIdWalletJournal200Ok)
			existingIDs, err := s.st.ListCharacterWalletJournalEntryIDs(ctx, characterID)
			if err != nil {
				return err
			}
			var newEntries []esi.GetCharactersCharacterIdWalletJournal200Ok
			for _, e := range entries {
				if existingIDs.Contains(e.Id) {
					continue
				}
				newEntries = append(newEntries, e)
			}
			slog.Debug("wallet journal", "existing", existingIDs, "entries", entries)
			if len(newEntries) == 0 {
				slog.Info("No new wallet journal entries", "characterID", characterID)
				return nil
			}
			ids := set.Of[int32]()
			for _, e := range newEntries {
				if e.FirstPartyId != 0 {
					ids.Add(e.FirstPartyId)
				}
				if e.SecondPartyId != 0 {
					ids.Add(e.SecondPartyId)
				}
				if e.TaxReceiverId != 0 {
					ids.Add(e.TaxReceiverId)
				}
			}
			_, err = s.eus.AddMissingEntities(ctx, ids)
			if err != nil {
				return err
			}
			for _, o := range newEntries {
				arg := storage.CreateCharacterWalletJournalEntryParams{
					Amount:        o.Amount,
					Balance:       o.Balance,
					ContextID:     o.ContextId,
					ContextIDType: o.ContextIdType,
					Date:          o.Date,
					Description:   o.Description,
					FirstPartyID:  o.FirstPartyId,
					RefID:         o.Id,
					CharacterID:   characterID,
					RefType:       o.RefType,
					Reason:        o.Reason,
					SecondPartyID: o.SecondPartyId,
					Tax:           o.Tax,
					TaxReceiverID: o.TaxReceiverId,
				}
				if err := s.st.CreateCharacterWalletJournalEntry(ctx, arg); err != nil {
					return err
				}
			}
			slog.Info("Stored new wallet journal entries", "characterID", characterID, "entries", len(newEntries))
			return nil
		})
}

const (
	maxTransactionsPerPage = 2_500 // maximum objects returned per page
)

func (s *CharacterService) ListWalletTransactions(ctx context.Context, characterID int32) ([]*app.CharacterWalletTransaction, error) {
	return s.st.ListCharacterWalletTransactions(ctx, characterID)
}

// updateWalletTransactionESI updates the wallet journal from ESI and reports whether it has changed.
func (s *CharacterService) updateWalletTransactionESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionWalletTransactions {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			transactions, err := s.fetchWalletTransactionsESI(ctx, characterID, arg.MaxWalletTransactions)
			if err != nil {
				return false, err
			}
			return transactions, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			transactions := data.([]esi.GetCharactersCharacterIdWalletTransactions200Ok)
			existingIDs, err := s.st.ListCharacterWalletTransactionIDs(ctx, characterID)
			if err != nil {
				return err
			}
			var newEntries []esi.GetCharactersCharacterIdWalletTransactions200Ok
			for _, e := range transactions {
				if existingIDs.Contains(e.TransactionId) {
					continue
				}
				newEntries = append(newEntries, e)
			}
			slog.Debug("wallet transaction", "existing", existingIDs, "entries", transactions)
			if len(newEntries) == 0 {
				slog.Info("No new wallet transactions", "characterID", characterID)
				return nil
			}
			var entityIDs, typeIDs set.Set[int32]
			var locationIDs set.Set[int64]
			for _, en := range newEntries {
				if en.ClientId != 0 {
					entityIDs.Add(en.ClientId)
				}
				locationIDs.Add(en.LocationId)
				typeIDs.Add(en.TypeId)
			}
			g := new(errgroup.Group)
			g.Go(func() error {
				_, err := s.eus.AddMissingEntities(ctx, entityIDs)
				return err
			})
			g.Go(func() error {
				return s.eus.AddMissingLocations(ctx, locationIDs)
			})
			g.Go(func() error {
				return s.eus.AddMissingTypes(ctx, typeIDs)
			})
			if err := g.Wait(); err != nil {
				return err
			}
			for _, o := range newEntries {
				arg := storage.CreateCharacterWalletTransactionParams{
					ClientID:      o.ClientId,
					Date:          o.Date,
					EveTypeID:     o.TypeId,
					IsBuy:         o.IsBuy,
					IsPersonal:    o.IsPersonal,
					JournalRefID:  o.JournalRefId,
					LocationID:    o.LocationId,
					CharacterID:   characterID,
					Quantity:      o.Quantity,
					TransactionID: o.TransactionId,
					UnitPrice:     o.UnitPrice,
				}
				if err := s.st.CreateCharacterWalletTransaction(ctx, arg); err != nil {
					return err
				}
			}
			slog.Info("Stored new wallet transactions", "characterID", characterID, "entries", len(newEntries))
			return nil
		})
}

// fetchWalletTransactionsESI fetches wallet transactions from ESI with paging and returns them.
func (s *CharacterService) fetchWalletTransactionsESI(ctx context.Context, characterID int32, maxTransactions int) ([]esi.GetCharactersCharacterIdWalletTransactions200Ok, error) {
	var oo2 []esi.GetCharactersCharacterIdWalletTransactions200Ok
	lastID := int64(0)
	for {
		var opts *esi.GetCharactersCharacterIdWalletTransactionsOpts
		if lastID > 0 {
			opts = &esi.GetCharactersCharacterIdWalletTransactionsOpts{FromId: esioptional.NewInt64(lastID)}
		} else {
			opts = nil
		}
		oo, _, err := s.esiClient.ESI.WalletApi.GetCharactersCharacterIdWalletTransactions(ctx, characterID, opts)
		if err != nil {
			return nil, err
		}
		oo2 = slices.Concat(oo2, oo)
		isLimitExceeded := (maxTransactions != 0 && len(oo2)+maxTransactionsPerPage > maxTransactions)
		if len(oo) < maxTransactionsPerPage || isLimitExceeded {
			break
		}
		ids := make([]int64, len(oo))
		for i, o := range oo {
			ids[i] = o.TransactionId
		}
		lastID = slices.Min(ids)
	}
	slog.Debug("Received wallet transactions", "characterID", characterID, "count", len(oo2))
	return oo2, nil
}

// fetchFromESIWithPaging returns the combined list of items from all pages of an ESI endpoint.
// This only works for ESI endpoints which support the X-Pages pattern and return a list.
func fetchFromESIWithPaging[T any](fetch func(int) ([]T, *http.Response, error)) ([]T, error) {
	result, r, err := fetch(1)
	if err != nil {
		return nil, err
	}
	pages, err := extractPageCount(r)
	if err != nil {
		return nil, err
	}
	if pages < 2 {
		return result, nil
	}
	results := make([][]T, pages)
	results[0] = result
	g := new(errgroup.Group)
	for p := 2; p <= pages; p++ {
		p := p
		g.Go(func() error {
			result, _, err := fetch(p)
			if err != nil {
				return err
			}
			results[p-1] = result
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	combined := make([]T, 0)
	for _, result := range results {
		combined = slices.Concat(combined, result)
	}
	return combined, nil
}

func extractPageCount(r *http.Response) (int, error) {
	x := r.Header.Get("X-Pages")
	if x == "" {
		return 1, nil
	}
	pages, err := strconv.Atoi(x)
	if err != nil {
		return 0, err
	}
	return pages, nil
}

// UpdateSectionIfNeeded updates a section from ESI if has expired and changed
// and reports back if it has changed
func (s *CharacterService) UpdateSectionIfNeeded(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.CharacterID == 0 || arg.Section == "" {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	if !arg.ForceUpdate {
		status, err := s.st.GetCharacterSectionStatus(ctx, arg.CharacterID, arg.Section)
		if err != nil {
			if !errors.Is(err, app.ErrNotFound) {
				return false, err
			}
		} else {
			if !status.HasError() && !status.IsExpired() {
				return false, nil
			}
		}
	}
	var f func(context.Context, app.CharacterUpdateSectionParams) (bool, error)
	switch arg.Section {
	case app.SectionAssets:
		f = s.updateAssetsESI
	case app.SectionAttributes:
		f = s.updateAttributesESI
	case app.SectionContracts:
		f = s.updateContractsESI
	case app.SectionImplants:
		f = s.updateImplantsESI
	case app.SectionIndustryJobs:
		f = s.updateIndustryJobsESI
	case app.SectionJumpClones:
		f = s.updateJumpClonesESI
	case app.SectionLocation:
		f = s.updateLocationESI
	case app.SectionMails:
		f = s.updateMailsESI
	case app.SectionMailLabels:
		f = s.updateMailLabelsESI
	case app.SectionMailLists:
		f = s.updateMailListsESI
	case app.SectionNotifications:
		f = s.updateNotificationsESI
	case app.SectionOnline:
		f = s.updateOnlineESI
	case app.SectionRoles:
		f = s.updateRolesESI
	case app.SectionPlanets:
		f = s.updatePlanetsESI
	case app.SectionShip:
		f = s.updateShipESI
	case app.SectionSkillqueue:
		f = s.UpdateSkillqueueESI
	case app.SectionSkills:
		f = s.updateSkillsESI
	case app.SectionWalletBalance:
		f = s.updateWalletBalanceESI
	case app.SectionWalletJournal:
		f = s.updateWalletJournalEntryESI
	case app.SectionWalletTransactions:
		f = s.updateWalletTransactionESI
	default:
		return false, fmt.Errorf("update section: unknown section: %s", arg.Section)
	}
	key := fmt.Sprintf("update-character-section-%s-%d", arg.Section, arg.CharacterID)
	x, err, _ := s.sfg.Do(key, func() (any, error) {
		return f(ctx, arg)
	})
	if err != nil {
		errorMessage := err.Error()
		startedAt := optional.Optional[time.Time]{}
		arg2 := storage.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID:  arg.CharacterID,
			Section:      arg.Section,
			ErrorMessage: &errorMessage,
			StartedAt:    &startedAt,
		}
		o, err2 := s.st.UpdateOrCreateCharacterSectionStatus(ctx, arg2)
		if err2 != nil {
			slog.Error("record error for failed section update: %s", "error", err2)
		}
		s.scs.SetCharacterSection(o)
		return false, fmt.Errorf("update character section from ESI for %v: %w", arg, err)
	}
	changed := x.(bool)
	slog.Info("Character section update completed", "characterID", arg.CharacterID, "section", arg.Section, "forced", arg.ForceUpdate, "changed", changed)
	return changed, err
}

// updateSectionIfChanged updates a character section if it has changed
// and reports whether it has changed
func (s *CharacterService) updateSectionIfChanged(
	ctx context.Context,
	arg app.CharacterUpdateSectionParams,
	fetch func(ctx context.Context, characterID int32) (any, error),
	update func(ctx context.Context, characterID int32, data any) error,
) (bool, error) {
	startedAt := optional.From(time.Now())
	arg2 := storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID: arg.CharacterID,
		Section:     arg.Section,
		StartedAt:   &startedAt,
	}
	o, err := s.st.UpdateOrCreateCharacterSectionStatus(ctx, arg2)
	if err != nil {
		return false, err
	}
	s.scs.SetCharacterSection(o)
	token, err := s.GetValidCharacterToken(ctx, arg.CharacterID)
	if err != nil {
		return false, err
	}
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, token.AccessToken)
	data, err := fetch(ctx, arg.CharacterID)
	if err != nil {
		return false, err
	}
	hash, err := calcContentHash(data)
	if err != nil {
		return false, err
	}

	// identify if changed
	var notFound bool
	u, err := s.st.GetCharacterSectionStatus(ctx, arg.CharacterID, arg.Section)
	if errors.Is(err, app.ErrNotFound) {
		notFound = true
	} else if err != nil {
		return false, err
	}

	// update if needed
	hasChanged := u.ContentHash != hash
	if arg.ForceUpdate || notFound || hasChanged {
		if err := update(ctx, arg.CharacterID, data); err != nil {
			return false, err
		}
	}

	// record successful completion
	completedAt := storage.NewNullTimeFromTime(time.Now())
	errorMessage := ""
	startedAt2 := optional.Optional[time.Time]{}
	arg2 = storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID: arg.CharacterID,
		Section:     arg.Section,

		ErrorMessage: &errorMessage,
		ContentHash:  &hash,
		CompletedAt:  &completedAt,
		StartedAt:    &startedAt2,
	}
	o, err = s.st.UpdateOrCreateCharacterSectionStatus(ctx, arg2)
	if err != nil {
		return false, err
	}
	s.scs.SetCharacterSection(o)
	slog.Debug("Has section changed", "characterID", arg.CharacterID, "section", arg.Section, "changed", hasChanged)
	return hasChanged, nil
}

func calcContentHash(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	b2 := md5.Sum(b)
	hash := hex.EncodeToString(b2[:])
	return hash, nil
}
