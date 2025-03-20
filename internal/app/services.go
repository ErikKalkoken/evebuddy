package app

import (
	"context"
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"

	"github.com/ErikKalkoken/evebuddy/internal/github"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

// Defines a cache service
type CacheService interface {
	Clear()
	Delete(string)
	Exists(string) bool
	Get(string) (any, bool)
	Set(string, any, time.Duration)
}

type EveImageService interface {
	AllianceLogo(int32, int) (fyne.Resource, error)
	CharacterPortrait(int32, int) (fyne.Resource, error)
	CorporationLogo(int32, int) (fyne.Resource, error)
	ClearCache() error
	EntityIcon(int32, string, int) (fyne.Resource, error)
	InventoryTypeBPO(int32, int) (fyne.Resource, error)
	InventoryTypeBPC(int32, int) (fyne.Resource, error)
	InventoryTypeIcon(int32, int) (fyne.Resource, error)
	InventoryTypeRender(int32, int) (fyne.Resource, error)
	InventoryTypeSKIN(int32, int) (fyne.Resource, error)
}

type EveUniverseService interface {
	AddMissingEntities(context.Context, []int32) ([]int32, error)
	AddMissingEveTypes(context.Context, []int32) error
	FormatDogmaValue(context.Context, float32, EveUnitID) (string, int32)
	GetAllianceCorporationsESI(context.Context, int32) ([]*EveEntity, error)
	GetAllianceESI(context.Context, int32) (*EveAlliance, error)
	GetCharacterCorporationHistory(context.Context, int32) ([]MembershipHistoryItem, error)
	GetCharacterESI(context.Context, int32) (*EveCharacter, error)
	GetConstellationSolarSytemsESI(context.Context, int32) ([]*EveSolarSystem, error)
	GetCorporationAllianceHistory(context.Context, int32) ([]MembershipHistoryItem, error)
	GetCorporationESI(context.Context, int32) (*EveCorporation, error)
	GetLocation(context.Context, int64) (*EveLocation, error)
	GetMarketPrice(context.Context, int32) (*EveMarketPrice, error)
	GetOrCreateCharacterESI(context.Context, int32) (*EveCharacter, error)
	GetOrCreateConstellationESI(context.Context, int32) (*EveConstellation, error)
	GetOrCreateEntityESI(context.Context, int32) (*EveEntity, error)
	GetOrCreateLocationESI(context.Context, int64) (*EveLocation, error)
	GetOrCreatePlanetESI(context.Context, int32) (*EvePlanet, error)
	GetOrCreateRegionESI(context.Context, int32) (*EveRegion, error)
	GetOrCreateSchematicESI(context.Context, int32) (*EveSchematic, error)
	GetOrCreateSolarSystemESI(context.Context, int32) (*EveSolarSystem, error)
	GetOrCreateTypeESI(context.Context, int32) (*EveType, error)
	GetPlanets(context.Context, []EveSolarSystemPlanet) ([]*EvePlanet, error)
	GetRegionConstellationsESI(context.Context, int32) ([]*EveEntity, error)
	GetSolarSystemInfoESI(ctx context.Context, solarSystemID int32) (int32, []EveSolarSystemPlanet, []int32, []*EveEntity, []*EveLocation, error)
	GetSolarSystemsESI(context.Context, []int32) ([]*EveSolarSystem, error)
	GetStarTypeID(context.Context, int32) (int32, error)
	GetType(context.Context, int32) (*EveType, error)
	ListEntitiesByPartialName(context.Context, string) ([]*EveEntity, error)
	ListLocations(context.Context) ([]*EveLocation, error)
	ListTypeDogmaAttributesForType(context.Context, int32) ([]*EveTypeDogmaAttribute, error)
	ToEveEntities(context.Context, []int32) (map[int32]*EveEntity, error)
	UpdateSection(context.Context, GeneralSection, bool) (bool, error)
}

// A service for fetching the current ESI Status.
type ESIStatusService interface {
	Fetch(context.Context) (*ESIStatus, error)
}

// CharacterService ...
type CharacterService interface {
	AddEveEntitiesFromCharacterSearchESI(ctx context.Context, characterID int32, search string) ([]int32, error)
	CharacterAssetTotalValue(ctx context.Context, characterID int32) (optional.Optional[float64], error)
	CharacterHasTokenWithScopes(ctx context.Context, characterID int32) (bool, error)
	CountCharacterContractBids(ctx context.Context, contractID int64) (int, error)
	CountCharacterNotificationUnreads(ctx context.Context, characterID int32) (map[NotificationGroup]int, error)
	DeleteCharacter(ctx context.Context, id int32) error
	DeleteCharacterMail(ctx context.Context, characterID, mailID int32) error
	DisableAllTrainingWatchers(ctx context.Context) error
	EnableAllTrainingWatchers(ctx context.Context) error
	EnableTrainingWatcher(ctx context.Context, characterID int32) error
	GetAllCharacterMailUnreadCount(ctx context.Context) (int, error)
	GetAnyCharacter(ctx context.Context) (*Character, error)
	GetCharacter(ctx context.Context, id int32) (*Character, error)
	GetCharacterAttributes(ctx context.Context, characterID int32) (*CharacterAttributes, error)
	GetCharacterContractTopBid(ctx context.Context, contractID int64) (*CharacterContractBid, error)
	GetCharacterMail(ctx context.Context, characterID int32, mailID int32) (*CharacterMail, error)
	GetCharacterMailCounts(ctx context.Context, characterID int32) (int, int, error)
	GetCharacterMailLabelUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error)
	GetCharacterMailListUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error)
	GetCharacterSkill(ctx context.Context, characterID, typeID int32) (*CharacterSkill, error)
	GetCharacterTotalTrainingTime(ctx context.Context, characterID int32) (optional.Optional[time.Duration], error)
	ListAllCharacterAssets(ctx context.Context) ([]*CharacterAsset, error)
	ListAllCharacterPlanets(ctx context.Context) ([]*CharacterPlanet, error)
	ListCharacterAssets(ctx context.Context, characterID int32) ([]*CharacterAsset, error)
	ListCharacterAssetsInItemHangar(ctx context.Context, characterID int32, locationID int64) ([]*CharacterAsset, error)
	ListCharacterAssetsInLocation(ctx context.Context, characterID int32, locationID int64) ([]*CharacterAsset, error)
	ListCharacterAssetsInShipHangar(ctx context.Context, characterID int32, locationID int64) ([]*CharacterAsset, error)
	ListCharacterContractItems(ctx context.Context, contractID int64) ([]*CharacterContractItem, error)
	ListCharacterContracts(ctx context.Context, characterID int32) ([]*CharacterContract, error)
	ListCharacterImplants(ctx context.Context, characterID int32) ([]*CharacterImplant, error)
	ListCharacterJumpClones(ctx context.Context, characterID int32) ([]*CharacterJumpClone, error)
	ListCharacterMailHeadersForLabelOrdered(ctx context.Context, characterID int32, labelID int32) ([]*CharacterMailHeader, error)
	ListCharacterMailHeadersForListOrdered(ctx context.Context, characterID int32, listID int32) ([]*CharacterMailHeader, error)
	ListCharacterMailLabelsOrdered(ctx context.Context, characterID int32) ([]*CharacterMailLabel, error)
	ListCharacterMailLists(ctx context.Context, characterID int32) ([]*EveEntity, error)
	ListCharacterNotificationsAll(ctx context.Context, characterID int32) ([]*CharacterNotification, error)
	ListCharacterNotificationsTypes(ctx context.Context, characterID int32, ng NotificationGroup) ([]*CharacterNotification, error) // TODO: Rename to ..Group
	ListCharacterNotificationsUnread(ctx context.Context, characterID int32) ([]*CharacterNotification, error)
	ListCharacterPlanets(ctx context.Context, characterID int32) ([]*CharacterPlanet, error)
	ListCharacters(ctx context.Context) ([]*Character, error)
	ListCharacterShipsAbilities(ctx context.Context, characterID int32, search string) ([]*CharacterShipAbility, error)
	ListCharacterSkillGroupsProgress(ctx context.Context, characterID int32) ([]ListCharacterSkillGroupProgress, error)
	ListCharacterSkillProgress(ctx context.Context, characterID, eveGroupID int32) ([]ListCharacterSkillProgress, error)
	ListCharacterSkillqueueItems(ctx context.Context, characterID int32) ([]*CharacterSkillqueueItem, error)
	ListCharactersShort(ctx context.Context) ([]*CharacterShort, error)
	ListCharacterWalletJournalEntries(ctx context.Context, characterID int32) ([]*CharacterWalletJournalEntry, error)
	ListCharacterWalletTransactions(ctx context.Context, characterID int32) ([]*CharacterWalletTransaction, error)
	NotifyCommunications(ctx context.Context, characterID int32, earliest time.Time, typesEnabled set.Set[string], notify func(title, content string)) error
	NotifyExpiredExtractions(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error
	NotifyExpiredTraining(ctx context.Context, characterID int32, notify func(title, content string)) error
	NotifyMails(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error
	NotifyUpdatedContracts(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error
	SearchESI(ctx context.Context, characterID int32, search string, categories []SearchCategory, strict bool) (map[SearchCategory][]*EveEntity, int, error)
	SendCharacterMail(ctx context.Context, characterID int32, subject string, recipients []*EveEntity, body string) (int32, error)
	UpdateCharacterAssetTotalValue(ctx context.Context, characterID int32) (float64, error)
	UpdateCharacterSkillqueueESI(ctx context.Context, arg CharacterUpdateSectionParams) (bool, error)
	UpdateMailRead(ctx context.Context, characterID, mailID int32) error
	UpdateOrCreateCharacterFromSSO(ctx context.Context, infoText binding.ExternalString) (int32, error)
	UpdateSectionIfNeeded(ctx context.Context, arg CharacterUpdateSectionParams) (bool, error)
}

type UI interface {
	AppName() string
	AvailableUpdate() (github.VersionInfo, error)
	CharacterService() CharacterService
	CurrentCharacter() *Character
	CurrentCharacterID() int32
	ESIStatusService() ESIStatusService
	EveImageService() EveImageService
	EveUniverseService() EveUniverseService
	HasCharacter() bool
	IsDesktop() bool
	IsDeveloperMode() bool
	IsMobile() bool
	IsOffline() bool
	MakeAboutPage() fyne.CanvasObject
	MakeCharacterSwitchMenu(refresh func()) []*fyne.MenuItem
	MakeWindowTitle(subTitle string) string
	MemCache() CacheService
	ModifyShortcutsForDialog(d dialog.Dialog, w fyne.Window)
	NewErrorDialog(message string, err error, parent fyne.Window) dialog.Dialog
	RefreshCrossPages()
	ResetCharacter()
	SetAnyCharacter() error
	ShowConfirmDialog(title, message, confirm string, callback func(bool), parent fyne.Window)
	ShowErrorDialog(message string, err error, parent fyne.Window)
	ShowEveEntityInfoWindow(o *EveEntity)
	ShowInformationDialog(title, message string, parent fyne.Window)
	ShowInfoWindow(c EveEntityCategory, id int32)
	ShowLocationInfoWindow(id int64)
	ShowTypeInfoWindow(id int32)
	ShowUpdateStatusWindow()
	StatusCacheService() StatusCacheService
	UpdateAvatar(id int32, setIcon func(fyne.Resource))
	UpdateCharacter()
	UpdateCharacterAndRefreshIfNeeded(ctx context.Context, characterID int32, forceUpdate bool)
	UpdateCharacterSectionAndRefreshIfNeeded(ctx context.Context, characterID int32, s CharacterSection, forceUpdate bool)
	UpdateGeneralSectionAndRefreshIfNeeded(ctx context.Context, section GeneralSection, forceUpdate bool)
	UpdateGeneralSectionsAndRefreshIfNeeded(forceUpdate bool)
	UpdateMailIndicator()
	UpdateStatus()
	WebsiteRootURL() *url.URL
}
