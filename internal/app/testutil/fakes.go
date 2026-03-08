package testutil

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/eveauth"
	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type AuthClientFake struct {
	Token *eveauth.Token
	Err   error
}

func (s AuthClientFake) Authorize(ctx context.Context, scopes []string) (*eveauth.Token, error) {
	return s.Token, s.Err
}

func (s AuthClientFake) RefreshToken(ctx context.Context, token *eveauth.Token) error {
	token.AccessToken = s.Token.AccessToken
	token.RefreshToken = s.Token.RefreshToken
	token.ExpiresAt = s.Token.ExpiresAt
	return nil
}

func AuthTokenFromAppToken(x *app.Token) *eveauth.Token {
	return &eveauth.Token{
		AccessToken:   x.AccessToken,
		CharacterID:   int32(x.CharacterID),
		CharacterName: x.CharacterName,
		ExpiresAt:     x.ExpiresAt,
		RefreshToken:  x.RefreshToken,
		Scopes:        slices.Clone(x.Scopes),
		TokenType:     x.TokenType,
	}
}

type CacheFake map[string][]byte

func NewCacheFake() CacheFake {
	return make(CacheFake)
}

func (c CacheFake) Get(k string) ([]byte, bool) {
	v, ok := c[k]
	return v, ok
}

func (c CacheFake) Set(k string, v []byte, d time.Duration) {
	c[k] = v
}

func (c CacheFake) Clear() {
	for k := range c {
		delete(c, k)
	}
}

type CacheFake2 map[string]any

func NewCacheFake2() CacheFake2 {
	return make(CacheFake2)
}

func (c CacheFake2) Delete(k string) {
	delete(c, k)
}

func (c CacheFake2) GetInt64(k string) (int64, bool) {
	v, ok := c[k]
	if !ok {
		return 0, false
	}
	return v.(int64), true
}

func (c CacheFake2) GetString(k string) (string, bool) {
	v, ok := c[k]
	if !ok {
		return "", false
	}
	return v.(string), true
}

func (c CacheFake2) SetInt64(k string, v int64, d time.Duration) {
	c[k] = v
}

func (c CacheFake2) SetString(k string, v string, d time.Duration) {
	c[k] = v
}

type EveUniverseServiceFake struct{}

func (s *EveUniverseServiceFake) GetOrCreateEntityESI(ctx context.Context, id int64) (*app.EveEntity, error) {
	o := &app.EveEntity{
		ID:       id,
		Name:     fmt.Sprintf("Entity%d", id),
		Category: app.EveEntityUnknown,
	}
	return o, nil
}

func (s *EveUniverseServiceFake) GetOrCreateLocationESI(ctx context.Context, id int64) (*app.EveLocation, error) {
	ss, _ := s.GetOrCreateSolarSystemESI(ctx, 30002537)
	owner, _ := s.GetOrCreateEntityESI(ctx, 42)
	o := &app.EveLocation{
		ID:          id,
		Name:        fmt.Sprintf("Location%d", id),
		Owner:       optional.New(owner),
		SolarSystem: optional.New(ss),
		UpdatedAt:   time.Now(),
		Type: optional.New(&app.EveType{
			ID:   35835,
			Name: fmt.Sprintf("Type%d", id),
			Group: &app.EveGroup{
				ID:   1406,
				Name: "Refinery",
				Category: &app.EveCategory{
					ID:   65,
					Name: "Structure",
				},
			},
		}),
	}
	return o, nil
}

func (s *EveUniverseServiceFake) GetOrCreateMoonESI(ctx context.Context, id int64) (*app.EveMoon, error) {
	ss, nil := s.GetOrCreateSolarSystemESI(ctx, 30002537)
	o := &app.EveMoon{
		ID:          id,
		Name:        fmt.Sprintf("Moon%d", id),
		SolarSystem: ss,
	}
	return o, nil
}

func (s *EveUniverseServiceFake) GetOrCreatePlanetESI(ctx context.Context, id int64) (*app.EvePlanet, error) {
	ss, _ := s.GetOrCreateSolarSystemESI(ctx, 30002537)
	et, _ := s.GetOrCreateTypeESI(ctx, 5)
	o := &app.EvePlanet{
		ID:          id,
		Name:        fmt.Sprintf("Planet%d", id),
		SolarSystem: ss,
		Type:        et,
	}
	return o, nil
}

func (s *EveUniverseServiceFake) GetOrCreateSolarSystemESI(ctx context.Context, id int64) (*app.EveSolarSystem, error) {
	o := &app.EveSolarSystem{
		ID:   id,
		Name: fmt.Sprintf("System%d", id),
		Constellation: &app.EveConstellation{
			ID:   20000372,
			Name: "Constellation",
			Region: &app.EveRegion{
				ID:   10000030,
				Name: "Region",
			},
		},
	}
	return o, nil
}

func (s *EveUniverseServiceFake) GetOrCreateTypeESI(ctx context.Context, id int64) (*app.EveType, error) {
	o := &app.EveType{
		ID:   id,
		Name: fmt.Sprintf("Type%d", id),
		Group: &app.EveGroup{
			ID:   420,
			Name: "Group",
			Category: &app.EveCategory{
				ID:   5,
				Name: "Category",
			},
		},
	}
	return o, nil
}

func (s *EveUniverseServiceFake) ToEntities(ctx context.Context, ids set.Set[int64]) (map[int64]*app.EveEntity, error) {
	m := make(map[int64]*app.EveEntity)
	for id := range ids.All() {
		o, _ := s.GetOrCreateEntityESI(ctx, id)
		m[id] = o
	}
	return m, nil
}

type EveImageServiceFake struct {
	Alliance    fyne.Resource
	Character   fyne.Resource
	Corporation fyne.Resource
	Err         error
	Faction     fyne.Resource
	Type        fyne.Resource
}

func (s *EveImageServiceFake) AllianceLogo(id int64, size int) (fyne.Resource, error) {
	return s.Alliance, s.Err
}

func (s *EveImageServiceFake) AllianceLogoAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Alliance)
}

func (s *EveImageServiceFake) CharacterPortrait(id int64, size int) (fyne.Resource, error) {
	return s.Character, s.Err
}

func (s *EveImageServiceFake) CharacterPortraitAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Character)
}

func (s *EveImageServiceFake) CorporationLogo(id int64, size int) (fyne.Resource, error) {
	return s.Corporation, s.Err
}

func (s *EveImageServiceFake) CorporationLogoAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Corporation)
}
func (s *EveImageServiceFake) FactionLogo(id int64, size int) (fyne.Resource, error) {
	return s.Faction, s.Err
}

func (s *EveImageServiceFake) FactionLogoAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Faction)
}

func (s *EveImageServiceFake) InventoryTypeRender(id int64, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceFake) InventoryTypeRenderAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Type)
}

func (s *EveImageServiceFake) InventoryTypeIcon(id int64, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceFake) InventoryTypeIconAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Type)
}

func (s *EveImageServiceFake) InventoryTypeBPO(id int64, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceFake) InventoryTypeBPOAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Type)
}

func (s *EveImageServiceFake) InventoryTypeBPC(id int64, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceFake) InventoryTypeBPCAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Type)
}
func (s *EveImageServiceFake) InventoryTypeSKIN(id int64, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceFake) InventoryTypeSKINAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Type)
}

type SettingsFake struct {
	MaxWalletTransactionsDefault    int
	MaxMailsDefault                 int
	MarketOrderRetentionDaysDefault int
}

func (s *SettingsFake) MaxMails() int {
	return s.MaxMailsDefault
}

func (s *SettingsFake) MaxWalletTransactions() int {
	return s.MaxWalletTransactionsDefault
}

func (s *SettingsFake) MarketOrderRetentionDays() int {
	return s.MarketOrderRetentionDaysDefault
}

func (s *SettingsFake) NotificationTypesEnabled() set.Set[string] {
	return set.Set[string]{}
}

func (s *SettingsFake) NotifyCommunicationsEarliest() time.Time {
	return time.Now()
}

func (s *SettingsFake) NotifyCommunicationsEnabled() bool {
	return true
}

func (s *SettingsFake) NotifyContractsEarliest() time.Time {
	return time.Now()
}

func (s *SettingsFake) NotifyContractsEnabled() bool {
	return true
}

func (s *SettingsFake) NotifyMailsEarliest() time.Time {
	return time.Now()
}
func (s *SettingsFake) NotifyMailsEnabled() bool {
	return true
}

func (s *SettingsFake) NotifyPIEnabled() bool {
	return true
}

func (s *SettingsFake) NotifyTrainingEnabled() bool {
	return true
}

func (s *SettingsFake) NotifyPIEarliest() time.Time {
	return time.Now()
}

func NewEveUniverseService(st *storage.Storage) *eveuniverseservice.EVEUniverseService {
	client := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
		UserAgent: "EveBuddy/1.0 (test@kalkoken.net)",
	})
	s := eveuniverseservice.New(eveuniverseservice.Params{
		ESIClient:          client,
		Signals:            app.NewSignals(),
		StatusCacheService: statuscacheservice.New(st),
		Storage:            st,
	})
	return s
}

// func NewX() *characterservice.CharacterService {

// }
