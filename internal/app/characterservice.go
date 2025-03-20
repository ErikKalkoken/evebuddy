package app

import (
	"context"
	"time"

	"fyne.io/fyne/v2/data/binding"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

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
