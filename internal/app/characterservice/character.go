// Package characterservice contains the EVE character service.
package characterservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

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
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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
	if arg.Section != app.SectionCharacterAssets {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	hasChanged, err := s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			assets, err := xesi.FetchWithPaging(
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
	if x := character.Corporation.IsNPC(); !x.IsEmpty() && !x.ValueOrZero() {
		setInfo("Fetching corporation from game server. Please wait...")
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
			if err := s.st.UpdateCharacterLocation(ctx, characterID, optional.From(locationID)); err != nil {
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
			if err := s.st.UpdateCharacterLastLoginAt(ctx, characterID, optional.From(online.LastLogin)); err != nil {
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
			if err := s.st.UpdateCharacterShip(ctx, characterID, optional.From(ship.ShipTypeId)); err != nil {
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
				content = fmt.Sprintf("Contract %s has been accepted by %s", name, c.AcceptorDisplay())
			case app.ContractStatusFinished:
				content = fmt.Sprintf("Contract %s has been delivered", name)
			case app.ContractStatusFailed:
				content = fmt.Sprintf("Contract %s has been failed by %s", name, c.AcceptorDisplay())
			}
		case app.ContractTypeItemExchange:
			switch c.Status {
			case app.ContractStatusFinished:
				content = fmt.Sprintf("Contract %s has been accepted by %s", name, c.AcceptorDisplay())
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
	"personal":    app.ContractAvailabilityPrivate,
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
	if arg.Section != app.SectionCharacterContracts {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			contracts, err := xesi.FetchWithPaging(
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
	if arg.Section != app.SectionCharacterImplants {
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

func (s *CharacterService) GetCharacterIndustryJob(ctx context.Context, characterID, jobID int32) (*app.CharacterIndustryJob, error) {
	return s.st.GetCharacterIndustryJob(ctx, characterID, jobID)
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
	if arg.Section != app.SectionCharacterIndustryJobs {
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
	if arg.Section != app.SectionCharacterJumpClones {
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
	if arg.Section != app.SectionCharacterNotifications {
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
	if arg.Section != app.SectionCharacterPlanets {
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
	if granted.Contains(app.RoleDirector) {
		roles = append(roles, app.CharacterRole{
			CharacterID: characterID,
			Role:        app.RoleDirector,
			Granted:     true,
		})
		return roles, nil
	}
	for _, r := range rolesSorted {
		roles = append(roles, app.CharacterRole{
			CharacterID: characterID,
			Role:        r,
			Granted:     granted.Contains(r),
		})
	}
	return roles, nil
}

// Roles
func (s *CharacterService) updateRolesESI(ctx context.Context, arg app.CharacterUpdateSectionParams) (bool, error) {
	if arg.Section != app.SectionCharacterRoles {
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
