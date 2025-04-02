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
	AddEveEntitiesFromSearchESI(ctx context.Context, characterID int32, search string) ([]int32, error)
	AssetTotalValue(ctx context.Context, characterID int32) (optional.Optional[float64], error)
	CountContractBids(ctx context.Context, contractID int64) (int, error)
	CountNotificationUnreads(ctx context.Context, characterID int32) (map[NotificationGroup]int, error)
	DeleteCharacter(ctx context.Context, id int32) error
	DeleteMail(ctx context.Context, characterID, mailID int32) error
	DisableAllTrainingWatchers(ctx context.Context) error
	EnableAllTrainingWatchers(ctx context.Context) error
	EnableTrainingWatcher(ctx context.Context, characterID int32) error
	GetAllMailUnreadCount(ctx context.Context) (int, error)
	GetAnyCharacter(ctx context.Context) (*Character, error)
	GetAttributes(ctx context.Context, characterID int32) (*CharacterAttributes, error)
	GetCharacter(ctx context.Context, id int32) (*Character, error)
	GetContractTopBid(ctx context.Context, contractID int64) (*CharacterContractBid, error)
	GetJumpClone(ctx context.Context, characterID, cloneID int32) (*CharacterJumpClone, error)
	GetMail(ctx context.Context, characterID int32, mailID int32) (*CharacterMail, error)
	GetMailCounts(ctx context.Context, characterID int32) (int, int, error)
	GetMailLabelUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error)
	GetMailListUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error)
	GetSkill(ctx context.Context, characterID, typeID int32) (*CharacterSkill, error)
	GetTotalTrainingTime(ctx context.Context, characterID int32) (optional.Optional[time.Duration], error)
	HasTokenWithScopes(ctx context.Context, characterID int32) (bool, error)
	ListAllAssets(ctx context.Context) ([]*CharacterAsset, error)
	ListAllJumpClones(ctx context.Context) ([]*CharacterJumpClone2, error)
	ListAllPlanets(ctx context.Context) ([]*CharacterPlanet, error)
	ListAssets(ctx context.Context, characterID int32) ([]*CharacterAsset, error)
	ListAssetsInItemHangar(ctx context.Context, characterID int32, locationID int64) ([]*CharacterAsset, error)
	ListAssetsInLocation(ctx context.Context, characterID int32, locationID int64) ([]*CharacterAsset, error)
	ListAssetsInShipHangar(ctx context.Context, characterID int32, locationID int64) ([]*CharacterAsset, error)
	ListCharacters(ctx context.Context) ([]*Character, error)
	ListCharactersShort(ctx context.Context) ([]*CharacterShort, error)
	ListContractItems(ctx context.Context, contractID int64) ([]*CharacterContractItem, error)
	ListContracts(ctx context.Context, characterID int32) ([]*CharacterContract, error)
	ListImplants(ctx context.Context, characterID int32) ([]*CharacterImplant, error)
	ListJumpClones(ctx context.Context, characterID int32) ([]*CharacterJumpClone, error)
	ListMailHeadersForLabelOrdered(ctx context.Context, characterID int32, labelID int32) ([]*CharacterMailHeader, error)
	ListMailHeadersForListOrdered(ctx context.Context, characterID int32, listID int32) ([]*CharacterMailHeader, error)
	ListMailLabelsOrdered(ctx context.Context, characterID int32) ([]*CharacterMailLabel, error)
	ListMailLists(ctx context.Context, characterID int32) ([]*EveEntity, error)
	ListNotificationsAll(ctx context.Context, characterID int32) ([]*CharacterNotification, error)
	ListNotificationsTypes(ctx context.Context, characterID int32, ng NotificationGroup) ([]*CharacterNotification, error)
	ListNotificationsUnread(ctx context.Context, characterID int32) ([]*CharacterNotification, error)
	ListPlanets(ctx context.Context, characterID int32) ([]*CharacterPlanet, error)
	ListShipsAbilities(ctx context.Context, characterID int32, search string) ([]*CharacterShipAbility, error)
	ListSkillGroupsProgress(ctx context.Context, characterID int32) ([]ListCharacterSkillGroupProgress, error)
	ListSkillProgress(ctx context.Context, characterID, eveGroupID int32) ([]ListSkillProgress, error)
	ListSkillqueueItems(ctx context.Context, characterID int32) ([]*CharacterSkillqueueItem, error)
	ListWalletJournalEntries(ctx context.Context, characterID int32) ([]*CharacterWalletJournalEntry, error)
	ListWalletTransactions(ctx context.Context, characterID int32) ([]*CharacterWalletTransaction, error)
	NotifyCommunications(ctx context.Context, characterID int32, earliest time.Time, typesEnabled set.Set[string], notify func(title, content string)) error
	NotifyExpiredExtractions(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error
	NotifyExpiredTraining(ctx context.Context, characterID int32, notify func(title, content string)) error
	NotifyMails(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error
	NotifyUpdatedContracts(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error
	SearchESI(ctx context.Context, characterID int32, search string, categories []SearchCategory, strict bool) (map[SearchCategory][]*EveEntity, int, error)
	SendMail(ctx context.Context, characterID int32, subject string, recipients []*EveEntity, body string) (int32, error)
	UpdateAssetTotalValue(ctx context.Context, characterID int32) (float64, error)
	UpdateIsTrainingWatched(ctx context.Context, id int32, v bool) error
	UpdateMailRead(ctx context.Context, characterID, mailID int32) error
	UpdateOrCreateCharacterFromSSO(ctx context.Context, infoText binding.ExternalString) (int32, error)
	UpdateSectionIfNeeded(ctx context.Context, arg CharacterUpdateSectionParams) (bool, error)
	UpdateSkillqueueESI(ctx context.Context, arg CharacterUpdateSectionParams) (bool, error)
}
