package testutil

import (
	"context"
	"fmt"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/ErikKalkoken/eveauth"
	"github.com/ErikKalkoken/go-set"
	"golang.org/x/oauth2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type AuthClientStub struct {
	Token *eveauth.Token
	Err   error
}

func (s AuthClientStub) Authorize(ctx context.Context, scopes []string) (*eveauth.Token, error) {
	return s.Token, s.Err
}

func (s AuthClientStub) RefreshToken(ctx context.Context, token *eveauth.Token) error {
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

type EUSEveNotificationServiceFake struct{}

func (s *EUSEveNotificationServiceFake) GetOrCreateEntityESI(ctx context.Context, id int64) (*app.EveEntity, error) {
	o := &app.EveEntity{
		ID:       id,
		Name:     fmt.Sprintf("Entity%d", id),
		Category: app.EveEntityUnknown,
	}
	return o, nil
}

func (s *EUSEveNotificationServiceFake) GetOrCreateLocationESI(ctx context.Context, id int64) (*app.EveLocation, error) {
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

func (s *EUSEveNotificationServiceFake) GetOrCreateMoonESI(ctx context.Context, id int64) (*app.EveMoon, error) {
	ss, err := s.GetOrCreateSolarSystemESI(ctx, 30002537)
	if err != nil {
		return nil, err
	}
	o := &app.EveMoon{
		ID:          id,
		Name:        fmt.Sprintf("Moon%d", id),
		SolarSystem: ss,
	}
	return o, nil
}

func (s *EUSEveNotificationServiceFake) GetOrCreatePlanetESI(ctx context.Context, id int64) (*app.EvePlanet, error) {
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

func (s *EUSEveNotificationServiceFake) GetOrCreateSolarSystemESI(ctx context.Context, id int64) (*app.EveSolarSystem, error) {
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

func (s *EUSEveNotificationServiceFake) GetOrCreateTypeESI(ctx context.Context, id int64) (*app.EveType, error) {
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

func (s *EUSEveNotificationServiceFake) ToEntities(ctx context.Context, ids set.Set[int64]) (map[int64]*app.EveEntity, error) {
	m := make(map[int64]*app.EveEntity)
	for id := range ids.All() {
		o, _ := s.GetOrCreateEntityESI(ctx, id)
		m[id] = o
	}
	return m, nil
}

type EveImageServiceStub struct {
	Alliance    fyne.Resource
	Character   fyne.Resource
	Corporation fyne.Resource
	Err         error
	Faction     fyne.Resource
	Type        fyne.Resource
}

func NewEveImageServiceStub() *EveImageServiceStub {
	s := &EveImageServiceStub{
		Character:   icons.Characterplaceholder64Jpeg,
		Alliance:    icons.Corporationplaceholder64Png,
		Corporation: icons.Corporationplaceholder64Png,
		Err:         nil,
		Faction:     icons.Factionplaceholder64Png,
		Type:        icons.Typeplaceholder64Png,
	}
	return s
}

func (s *EveImageServiceStub) AllianceLogo(id int64, size int) (fyne.Resource, error) {
	return s.Alliance, s.Err
}

func (s *EveImageServiceStub) AllianceLogoAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Alliance)
}

func (s *EveImageServiceStub) CharacterPortrait(id int64, size int) (fyne.Resource, error) {
	return s.Character, s.Err
}

func (s *EveImageServiceStub) CharacterPortraitAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Character)
}

func (s *EveImageServiceStub) CorporationLogo(id int64, size int) (fyne.Resource, error) {
	return s.Corporation, s.Err
}

func (s *EveImageServiceStub) CorporationLogoAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Corporation)
}

func (s *EveImageServiceStub) EveEntityLogoAsync(o *app.EveEntity, size int, setter func(r fyne.Resource)) {
	setter(s.eveEntityResource(o.Category))
}

func (s *EveImageServiceStub) EveEntityLogo(o *app.EveEntity, size int) (fyne.Resource, error) {
	return s.eveEntityResource(o.Category), nil
}

func (s *EveImageServiceStub) eveEntityResource(c app.EveEntityCategory) fyne.Resource {
	switch c {
	case app.EveEntityAlliance:
		return s.Alliance
	case app.EveEntityCharacter:
		return s.Character
	case app.EveEntityCorporation:
		return s.Corporation
	case app.EveEntityFaction:
		return s.Faction
	}
	return theme.BrokenImageIcon()
}

func (s *EveImageServiceStub) FactionLogo(id int64, size int) (fyne.Resource, error) {
	return s.Faction, s.Err
}

func (s *EveImageServiceStub) FactionLogoAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Faction)
}

func (s *EveImageServiceStub) InventoryTypeRender(id int64, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceStub) InventoryTypeRenderAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Type)
}

func (s *EveImageServiceStub) InventoryTypeIcon(id int64, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceStub) InventoryTypeIconAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Type)
}

func (s *EveImageServiceStub) InventoryTypeBPO(id int64, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceStub) InventoryTypeBPOAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Type)
}

func (s *EveImageServiceStub) InventoryTypeBPC(id int64, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceStub) InventoryTypeBPCAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Type)
}
func (s *EveImageServiceStub) InventoryTypeSKIN(id int64, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceStub) InventoryTypeSKINAsync(id int64, size int, setter func(r fyne.Resource)) {
	setter(s.Type)
}

type SettingsStub struct {
	MaxWalletTransactionsDefault    int
	MaxMailsDefault                 int
	MarketOrderRetentionDaysDefault int
}

func (s *SettingsStub) MaxMails() int {
	return s.MaxMailsDefault
}

func (s *SettingsStub) MaxWalletTransactions() int {
	return s.MaxWalletTransactionsDefault
}

func (s *SettingsStub) MarketOrderRetentionDays() int {
	return s.MarketOrderRetentionDaysDefault
}

func (s *SettingsStub) NotificationTypesEnabled() set.Set[string] {
	return set.Set[string]{}
}

func (s *SettingsStub) NotifyCommunicationsEarliest() time.Time {
	return time.Now()
}

func (s *SettingsStub) NotifyCommunicationsEnabled() bool {
	return true
}

func (s *SettingsStub) NotifyContractsEarliest() time.Time {
	return time.Now()
}

func (s *SettingsStub) NotifyContractsEnabled() bool {
	return true
}

func (s *SettingsStub) NotifyMailsEarliest() time.Time {
	return time.Now()
}
func (s *SettingsStub) NotifyMailsEnabled() bool {
	return true
}

func (s *SettingsStub) NotifyPIEnabled() bool {
	return true
}

func (s *SettingsStub) NotifyTrainingEnabled() bool {
	return true
}

func (s *SettingsStub) NotifyPIEarliest() time.Time {
	return time.Now()
}

type TokenSourceStub struct {
	CharacterToken *app.CharacterToken
	Error          error
}

func (ts TokenSourceStub) Token() (*oauth2.Token, error) {
	if ts.Error != nil {
		return nil, ts.Error
	}
	return ts.CharacterToken.OauthToken(), nil
}
