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
	DeleteCharacter(ctx context.Context, id int32) error
	// EnableTrainingWatcher enables training watcher for a character when it has an active training queue.
	EnableTrainingWatcher(ctx context.Context, characterID int32) error
	// EnableAllTrainingWatchers enables training watches for any currently active training queue.
	EnableAllTrainingWatchers(ctx context.Context) error
	// DisableAllTrainingWatchers disables training watches for all characters.
	DisableAllTrainingWatchers(ctx context.Context) error
	// GetCharacter returns a character from storage and updates calculated fields.
	GetCharacter(ctx context.Context, id int32) (*Character, error)
	GetAnyCharacter(ctx context.Context) (*Character, error)
	ListCharacters(ctx context.Context) ([]*Character, error)
	ListCharactersShort(ctx context.Context) ([]*CharacterShort, error)
	UpdateCharacterIsTrainingWatched(ctx context.Context, id int32, v bool) error
	// UpdateOrCreateCharacterFromSSO creates or updates a character via SSO authentication.
	// The provided context is used for the SSO authentication process only and can be canceled.
	UpdateOrCreateCharacterFromSSO(ctx context.Context, infoText binding.ExternalString) (int32, error)
	// AddEveEntitiesFromCharacterSearchESI runs a search on ESI and adds the results as new EveEntity objects to the database.
	// This method performs a character specific search and needs a token.
	AddEveEntitiesFromCharacterSearchESI(ctx context.Context, characterID int32, search string) ([]int32, error)
	ListCharacterAssetsInShipHangar(ctx context.Context, characterID int32, locationID int64) ([]*CharacterAsset, error)
	ListCharacterAssetsInItemHangar(ctx context.Context, characterID int32, locationID int64) ([]*CharacterAsset, error)
	ListCharacterAssetsInLocation(ctx context.Context, characterID int32, locationID int64) ([]*CharacterAsset, error)
	ListCharacterAssets(ctx context.Context, characterID int32) ([]*CharacterAsset, error)
	ListAllCharacterAssets(ctx context.Context) ([]*CharacterAsset, error)
	CharacterAssetTotalValue(ctx context.Context, characterID int32) (optional.Optional[float64], error)
	UpdateCharacterAssetTotalValue(ctx context.Context, characterID int32) (float64, error)
	GetCharacterAttributes(ctx context.Context, characterID int32) (*CharacterAttributes, error)
	CountCharacterContractBids(ctx context.Context, contractID int64) (int, error)
	GetCharacterContractTopBid(ctx context.Context, contractID int64) (*CharacterContractBid, error)
	NotifyUpdatedContracts(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error
	ListCharacterContracts(ctx context.Context, characterID int32) ([]*CharacterContract, error)
	ListCharacterContractItems(ctx context.Context, contractID int64) ([]*CharacterContractItem, error)
	ListCharacterImplants(ctx context.Context, characterID int32) ([]*CharacterImplant, error)
	GetCharacterJumpClone(ctx context.Context, characterID, cloneID int32) (*CharacterJumpClone, error)
	ListAllCharacterJumpClones(ctx context.Context) ([]*CharacterJumpClone2, error)
	ListCharacterJumpClones(ctx context.Context, characterID int32) ([]*CharacterJumpClone, error)
	// DeleteCharacterMail deletes a mail both on ESI and in the database.
	DeleteCharacterMail(ctx context.Context, characterID, mailID int32) error
	GetCharacterMail(ctx context.Context, characterID int32, mailID int32) (*CharacterMail, error)
	GetAllCharacterMailUnreadCount(ctx context.Context) (int, error)
	// GetCharacterMailCounts returns the number of unread mail for a character.
	GetCharacterMailCounts(ctx context.Context, characterID int32) (int, int, error)
	GetCharacterMailLabelUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error)
	GetCharacterMailListUnreadCounts(ctx context.Context, characterID int32) (map[int32]int, error)
	NotifyMails(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error
	ListCharacterMailLists(ctx context.Context, characterID int32) ([]*EveEntity, error)
	// ListMailsForLabel returns a character's mails for a label in descending order by timestamp.
	ListCharacterMailHeadersForLabelOrdered(ctx context.Context, characterID int32, labelID int32) ([]*CharacterMailHeader, error)
	ListCharacterMailHeadersForListOrdered(ctx context.Context, characterID int32, listID int32) ([]*CharacterMailHeader, error)
	ListCharacterMailLabelsOrdered(ctx context.Context, characterID int32) ([]*CharacterMailLabel, error)
	// SendCharacterMail creates a new mail on ESI and stores it locally.
	SendCharacterMail(ctx context.Context, characterID int32, subject string, recipients []*EveEntity, body string) (int32, error)
	// UpdateMailRead updates an existing mail as read
	UpdateMailRead(ctx context.Context, characterID, mailID int32) error
	CountCharacterNotificationUnreads(ctx context.Context, characterID int32) (map[NotificationGroup]int, error)
	NotifyCommunications(ctx context.Context, characterID int32, earliest time.Time, typesEnabled set.Set[string], notify func(title, content string)) error
	ListCharacterNotificationsTypes(ctx context.Context, characterID int32, ng NotificationGroup) ([]*CharacterNotification, error)
	ListCharacterNotificationsAll(ctx context.Context, characterID int32) ([]*CharacterNotification, error)
	ListCharacterNotificationsUnread(ctx context.Context, characterID int32) ([]*CharacterNotification, error)
	NotifyExpiredExtractions(ctx context.Context, characterID int32, earliest time.Time, notify func(title, content string)) error
	ListAllCharacterPlanets(ctx context.Context) ([]*CharacterPlanet, error)
	ListCharacterPlanets(ctx context.Context, characterID int32) ([]*CharacterPlanet, error)
	// UpdateSectionIfNeeded updates a section from ESI if has expired and changed
	// and reports back if it has changed
	UpdateSectionIfNeeded(ctx context.Context, arg CharacterUpdateSectionParams) (bool, error)
	ListCharacterShipsAbilities(ctx context.Context, characterID int32, search string) ([]*CharacterShipAbility, error)
	GetCharacterTotalTrainingTime(ctx context.Context, characterID int32) (optional.Optional[time.Duration], error)
	NotifyExpiredTraining(ctx context.Context, characterID int32, notify func(title, content string)) error
	ListCharacterSkillqueueItems(ctx context.Context, characterID int32) ([]*CharacterSkillqueueItem, error)
	// UpdateCharacterSkillqueueESI updates the skillqueue for a character from ESI
	// and reports wether it has changed.
	UpdateCharacterSkillqueueESI(ctx context.Context, arg CharacterUpdateSectionParams) (bool, error)
	GetCharacterSkill(ctx context.Context, characterID, typeID int32) (*CharacterSkill, error)
	ListCharacterSkillProgress(ctx context.Context, characterID, eveGroupID int32) ([]ListCharacterSkillProgress, error)
	ListCharacterSkillGroupsProgress(ctx context.Context, characterID int32) ([]ListCharacterSkillGroupProgress, error)
	// CharacterHasTokenWithScopes reports wether a token with the requested scopes exists for a character.
	CharacterHasTokenWithScopes(ctx context.Context, characterID int32) (bool, error)
	ListCharacterWalletJournalEntries(ctx context.Context, characterID int32) ([]*CharacterWalletJournalEntry, error)
	ListCharacterWalletTransactions(ctx context.Context, characterID int32) ([]*CharacterWalletTransaction, error)
	// SearchESI performs a name search for items on the ESI server
	// and returns the results by EveEntity category and sorted by name.
	// It also returns the total number of results.
	// A total of 500 indicates that we exceeded the server limit.
	SearchESI(ctx context.Context, characterID int32, search string, categories []SearchCategory, strict bool) (map[SearchCategory][]*EveEntity, int, error)
}
