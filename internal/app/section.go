package app

import (
	"log/slog"
	"strings"
	"time"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

const (
	characterSectionDefaultTimeout   = 3600 * time.Second
	corporationSectionDefaultTimeout = 3600 * time.Second
	generalSectionDefaultTimeout     = 24 * time.Hour
	sectionErrorTimeout              = 120 * time.Second
)

// section defines the interface for all section types.
type section interface {
	// DisplayName returns the output friendly name of a section.
	DisplayName() string

	// IsSkippingChangeDetection reports whether a section is skipping the change detection.
	IsSkippingChangeDetection() bool

	// Scopes returns the required scopes for fetching data for a section from ESI.
	Scopes() set.Set[string]

	// Implements the stringer interface.
	String() string

	// Timeout returns the time until the data of an update section becomes stale.
	Timeout() time.Duration
}

func makeSectionDisplayName(s section) string {
	t := strings.ReplaceAll(s.String(), "_", " ")
	t = xstrings.Title(t)
	return t
}

// CharacterSection represents a topic of a character that can be updated from ESI.
type CharacterSection string

var _ section = (*CharacterSection)(nil)

const (
	SectionCharacterAssets             CharacterSection = "assets"              // char-asset
	SectionCharacterAttributes         CharacterSection = "attributes"          // char-social
	SectionCharacterContracts          CharacterSection = "contracts"           // char-contract
	SectionCharacterImplants           CharacterSection = "implants"            // char-detail
	SectionCharacterIndustryJobs       CharacterSection = "industry_jobs"       // char-industry
	SectionCharacterJumpClones         CharacterSection = "jump_clones"         // char-location
	SectionCharacterLocation           CharacterSection = "location"            // char-location
	SectionCharacterMailLabels         CharacterSection = "mail_labels"         // char-social
	SectionCharacterMailLists          CharacterSection = "mail_lists"          // char-social
	SectionCharacterMailHeaders        CharacterSection = "mail_headers"        // char-social
	SectionCharacterMarketOrders       CharacterSection = "market_orders"       // char-market
	SectionCharacterNotifications      CharacterSection = "notifications"       // char-social
	SectionCharacterOnline             CharacterSection = "online"              // char-location
	SectionCharacterPlanets            CharacterSection = "planets"             // char-industry
	SectionCharacterRoles              CharacterSection = "roles"               // char-detail
	SectionCharacterShip               CharacterSection = "ship"                // char-location
	SectionCharacterSkillqueue         CharacterSection = "skillqueue"          // char-detail
	SectionCharacterSkills             CharacterSection = "skills"              // char-detail
	SectionCharacterWalletBalance      CharacterSection = "wallet_balance"      // char-wallet
	SectionCharacterWalletJournal      CharacterSection = "wallet_journal"      // char-wallet
	SectionCharacterWalletTransactions CharacterSection = "wallet_transactions" // char-wallet
)

var CharacterSections = []CharacterSection{
	SectionCharacterAssets,
	SectionCharacterAttributes,
	SectionCharacterContracts,
	SectionCharacterImplants,
	SectionCharacterIndustryJobs,
	SectionCharacterJumpClones,
	SectionCharacterLocation,
	SectionCharacterMailHeaders,
	SectionCharacterMailLabels,
	SectionCharacterMailLists,
	SectionCharacterMarketOrders,
	SectionCharacterNotifications,
	SectionCharacterOnline,
	SectionCharacterPlanets,
	SectionCharacterRoles,
	SectionCharacterShip,
	SectionCharacterSkillqueue,
	SectionCharacterSkills,
	SectionCharacterWalletBalance,
	SectionCharacterWalletJournal,
	SectionCharacterWalletTransactions,
}

func (cs CharacterSection) DisplayName() string {
	return makeSectionDisplayName(cs)
}

func (cs CharacterSection) IsSkippingChangeDetection() bool {
	switch cs {
	case SectionCharacterIndustryJobs:
		return true
	}
	return false
}

func (cs CharacterSection) Scopes() set.Set[string] {
	m := map[CharacterSection][]string{
		SectionCharacterAssets:             {"esi-assets.read_assets.v1", "esi-universe.read_structures.v1"},
		SectionCharacterAttributes:         {"esi-skills.read_skills.v1"},
		SectionCharacterContracts:          {"esi-contracts.read_character_contracts.v1", "esi-universe.read_structures.v1"},
		SectionCharacterImplants:           {"esi-clones.read_implants.v1"},
		SectionCharacterIndustryJobs:       {"esi-industry.read_character_jobs.v1", "esi-universe.read_structures.v1"},
		SectionCharacterJumpClones:         {"esi-clones.read_clones.v1", "esi-universe.read_structures.v1"},
		SectionCharacterLocation:           {"esi-location.read_location.v1", "esi-universe.read_structures.v1"},
		SectionCharacterMailHeaders:        {"esi-mail.organize_mail.v1", "esi-mail.read_mail.v1"},
		SectionCharacterMailLabels:         {"esi-mail.read_mail.v1"},
		SectionCharacterMailLists:          {"esi-mail.read_mail.v1"},
		SectionCharacterMarketOrders:       {"esi-markets.read_character_orders.v1"},
		SectionCharacterNotifications:      {"esi-characters.read_notifications.v1", "esi-universe.read_structures.v1"},
		SectionCharacterOnline:             {"esi-location.read_online.v1"},
		SectionCharacterPlanets:            {"esi-planets.manage_planets.v1"},
		SectionCharacterRoles:              {"esi-characters.read_corporation_roles.v1"},
		SectionCharacterShip:               {"esi-location.read_ship_type.v1"},
		SectionCharacterSkillqueue:         {"esi-skills.read_skillqueue.v1"},
		SectionCharacterSkills:             {"esi-skills.read_skills.v1"},
		SectionCharacterWalletBalance:      {"esi-wallet.read_character_wallet.v1"},
		SectionCharacterWalletJournal:      {"esi-wallet.read_character_wallet.v1"},
		SectionCharacterWalletTransactions: {"esi-wallet.read_character_wallet.v1", "esi-universe.read_structures.v1"},
	}
	scopes, ok := m[cs]
	if !ok {
		slog.Warn("Requested scopes for unknown section. Using default.", "section", cs)
		return set.Of[string]()
	}
	return set.Of(scopes...)
}

func (cs CharacterSection) String() string {
	return string(cs)
}

func (cs CharacterSection) Timeout() time.Duration {
	var m = map[CharacterSection]time.Duration{
		SectionCharacterAssets:             3600 * time.Second,
		SectionCharacterAttributes:         120 * time.Second,
		SectionCharacterContracts:          300 * time.Second,
		SectionCharacterImplants:           120 * time.Second,
		SectionCharacterIndustryJobs:       300 * time.Second,
		SectionCharacterJumpClones:         120 * time.Second,
		SectionCharacterLocation:           300 * time.Second, // minimum 5 seconds
		SectionCharacterMailLabels:         60 * time.Second,  // minimum 30 seconds
		SectionCharacterMailLists:          120 * time.Second,
		SectionCharacterMailHeaders:        60 * time.Second, // minimum 30 seconds
		SectionCharacterMarketOrders:       1200 * time.Second,
		SectionCharacterNotifications:      600 * time.Second,
		SectionCharacterOnline:             300 * time.Second, // minimum 30 seconds
		SectionCharacterPlanets:            600 * time.Second,
		SectionCharacterRoles:              3600 * time.Second,
		SectionCharacterShip:               300 * time.Second, // minimum 5 seconds
		SectionCharacterSkillqueue:         120 * time.Second,
		SectionCharacterSkills:             120 * time.Second,
		SectionCharacterWalletBalance:      120 * time.Second,
		SectionCharacterWalletJournal:      3600 * time.Second,
		SectionCharacterWalletTransactions: 3600 * time.Second,
	}
	duration, ok := m[cs]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", cs)
		return characterSectionDefaultTimeout
	}
	return duration
}

// CorporationSection represents a topic of a corporation that can be updated from ESI.
type CorporationSection string

var _ section = (*CorporationSection)(nil)

const (
	SectionCorporationAssets              CorporationSection = "assets"                // corp-assets
	SectionCorporationContracts           CorporationSection = "contracts"             // corp-contract
	SectionCorporationDivisions           CorporationSection = "divisions"             // corp-wallet
	SectionCorporationIndustryJobs        CorporationSection = "industry_jobs"         // corp-industry
	SectionCorporationMembers             CorporationSection = "members"               // corp-member
	SectionCorporationStructures          CorporationSection = "structures"            // corp-asset
	SectionCorporationWalletBalances      CorporationSection = "wallet_balances"       // corp-wallet
	SectionCorporationWalletJournal1      CorporationSection = "wallet_journal_1"      // corp-wallet
	SectionCorporationWalletJournal2      CorporationSection = "wallet_journal_2"      // corp-wallet
	SectionCorporationWalletJournal3      CorporationSection = "wallet_journal_3"      // corp-wallet
	SectionCorporationWalletJournal4      CorporationSection = "wallet_journal_4"      // corp-wallet
	SectionCorporationWalletJournal5      CorporationSection = "wallet_journal_5"      // corp-wallet
	SectionCorporationWalletJournal6      CorporationSection = "wallet_journal_6"      // corp-wallet
	SectionCorporationWalletJournal7      CorporationSection = "wallet_journal_7"      // corp-wallet
	SectionCorporationWalletTransactions1 CorporationSection = "wallet_transactions_1" // corp-wallet
	SectionCorporationWalletTransactions2 CorporationSection = "wallet_transactions_2" // corp-wallet
	SectionCorporationWalletTransactions3 CorporationSection = "wallet_transactions_3" // corp-wallet
	SectionCorporationWalletTransactions4 CorporationSection = "wallet_transactions_4" // corp-wallet
	SectionCorporationWalletTransactions5 CorporationSection = "wallet_transactions_5" // corp-wallet
	SectionCorporationWalletTransactions6 CorporationSection = "wallet_transactions_6" // corp-wallet
	SectionCorporationWalletTransactions7 CorporationSection = "wallet_transactions_7" // corp-wallet
)

var CorporationSections = []CorporationSection{
	SectionCorporationAssets,
	SectionCorporationContracts,
	SectionCorporationDivisions,
	SectionCorporationIndustryJobs,
	SectionCorporationMembers,
	SectionCorporationStructures,
	SectionCorporationWalletBalances,
	SectionCorporationWalletJournal1,
	SectionCorporationWalletJournal2,
	SectionCorporationWalletJournal3,
	SectionCorporationWalletJournal4,
	SectionCorporationWalletJournal5,
	SectionCorporationWalletJournal6,
	SectionCorporationWalletJournal7,
	SectionCorporationWalletTransactions1,
	SectionCorporationWalletTransactions2,
	SectionCorporationWalletTransactions3,
	SectionCorporationWalletTransactions4,
	SectionCorporationWalletTransactions5,
	SectionCorporationWalletTransactions6,
	SectionCorporationWalletTransactions7,
}

func CorporationSectionWalletJournal(d Division) CorporationSection {
	section := map[Division]CorporationSection{
		Division1: SectionCorporationWalletJournal1,
		Division2: SectionCorporationWalletJournal2,
		Division3: SectionCorporationWalletJournal3,
		Division4: SectionCorporationWalletJournal4,
		Division5: SectionCorporationWalletJournal5,
		Division6: SectionCorporationWalletJournal6,
		Division7: SectionCorporationWalletJournal7,
	}
	return section[d]
}

func CorporationSectionWalletTransactions(d Division) CorporationSection {
	section := map[Division]CorporationSection{
		Division1: SectionCorporationWalletTransactions1,
		Division2: SectionCorporationWalletTransactions2,
		Division3: SectionCorporationWalletTransactions3,
		Division4: SectionCorporationWalletTransactions4,
		Division5: SectionCorporationWalletTransactions5,
		Division6: SectionCorporationWalletTransactions6,
		Division7: SectionCorporationWalletTransactions7,
	}
	return section[d]
}

func (cs CorporationSection) DisplayName() string {
	return makeSectionDisplayName(cs)
}

// Division returns the division ID this section is related to or 0 if it has none.
func (cs CorporationSection) Division() Division {
	m := map[CorporationSection]Division{
		SectionCorporationWalletJournal1:      Division1,
		SectionCorporationWalletJournal2:      Division2,
		SectionCorporationWalletJournal3:      Division3,
		SectionCorporationWalletJournal4:      Division4,
		SectionCorporationWalletJournal5:      Division5,
		SectionCorporationWalletJournal6:      Division6,
		SectionCorporationWalletJournal7:      Division7,
		SectionCorporationWalletTransactions1: Division1,
		SectionCorporationWalletTransactions2: Division2,
		SectionCorporationWalletTransactions3: Division3,
		SectionCorporationWalletTransactions4: Division4,
		SectionCorporationWalletTransactions5: Division5,
		SectionCorporationWalletTransactions6: Division6,
		SectionCorporationWalletTransactions7: Division7,
	}
	return m[cs]
}

func (cs CorporationSection) IsSkippingChangeDetection() bool {
	switch cs {
	case SectionCorporationIndustryJobs:
		return true
	}
	return false
}

func (cs CorporationSection) String() string {
	return string(cs)
}

func (cs CorporationSection) Timeout() time.Duration {
	const (
		walletTransactions = 3600 * time.Second
		walletJournal      = 3600 * time.Second
	)
	m := map[CorporationSection]time.Duration{
		SectionCorporationAssets:              3600 * time.Second,
		SectionCorporationContracts:           300 * time.Second,
		SectionCorporationDivisions:           3600 * time.Second,
		SectionCorporationIndustryJobs:        300 * time.Second,
		SectionCorporationMembers:             3600 * time.Second,
		SectionCorporationWalletBalances:      300 * time.Second,
		SectionCorporationStructures:          3600 * time.Second,
		SectionCorporationWalletJournal1:      walletJournal,
		SectionCorporationWalletJournal2:      walletJournal,
		SectionCorporationWalletJournal3:      walletJournal,
		SectionCorporationWalletJournal4:      walletJournal,
		SectionCorporationWalletJournal5:      walletJournal,
		SectionCorporationWalletJournal6:      walletJournal,
		SectionCorporationWalletJournal7:      walletJournal,
		SectionCorporationWalletTransactions1: walletTransactions,
		SectionCorporationWalletTransactions2: walletTransactions,
		SectionCorporationWalletTransactions3: walletTransactions,
		SectionCorporationWalletTransactions4: walletTransactions,
		SectionCorporationWalletTransactions5: walletTransactions,
		SectionCorporationWalletTransactions6: walletTransactions,
		SectionCorporationWalletTransactions7: walletTransactions,
	}
	duration, ok := m[cs]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", cs)
		return corporationSectionDefaultTimeout
	}
	return duration
}

// Roles returns which roles are required for fetching data for a section from ESI.
func (cs CorporationSection) Roles() set.Set[Role] {
	var (
		anyAccountant = []Role{RoleAccountant, RoleJuniorAccountant}
	)
	m := map[CorporationSection][]Role{
		SectionCorporationAssets:              {RoleDirector},
		SectionCorporationContracts:           {},
		SectionCorporationDivisions:           {RoleDirector},
		SectionCorporationIndustryJobs:        {RoleFactoryManager},
		SectionCorporationMembers:             {},
		SectionCorporationStructures:          {RoleStationManager},
		SectionCorporationWalletBalances:      anyAccountant,
		SectionCorporationWalletJournal1:      anyAccountant,
		SectionCorporationWalletJournal2:      anyAccountant,
		SectionCorporationWalletJournal3:      anyAccountant,
		SectionCorporationWalletJournal4:      anyAccountant,
		SectionCorporationWalletJournal5:      anyAccountant,
		SectionCorporationWalletJournal6:      anyAccountant,
		SectionCorporationWalletJournal7:      anyAccountant,
		SectionCorporationWalletTransactions1: anyAccountant,
		SectionCorporationWalletTransactions2: anyAccountant,
		SectionCorporationWalletTransactions3: anyAccountant,
		SectionCorporationWalletTransactions4: anyAccountant,
		SectionCorporationWalletTransactions5: anyAccountant,
		SectionCorporationWalletTransactions6: anyAccountant,
		SectionCorporationWalletTransactions7: anyAccountant,
	}
	role, ok := m[cs]
	if !ok {
		slog.Warn("Requested role for unknown section. Using default.", "section", cs)
		return set.Of(RoleDirector)
	}
	return set.Of(role...)
}

func (cs CorporationSection) Scopes() set.Set[string] {
	journal := []string{"esi-wallet.read_corporation_wallets.v1"}
	transactions := []string{"esi-wallet.read_corporation_wallets.v1", "esi-universe.read_structures.v1"}
	m := map[CorporationSection][]string{
		SectionCorporationAssets:              {"esi-assets.read_corporation_assets.v1"},
		SectionCorporationContracts:           {"esi-contracts.read_corporation_contracts.v1"},
		SectionCorporationDivisions:           {"esi-corporations.read_divisions.v1"},
		SectionCorporationIndustryJobs:        {"esi-industry.read_corporation_jobs.v1"},
		SectionCorporationMembers:             {"esi-corporations.read_corporation_membership.v1"},
		SectionCorporationStructures:          {"esi-corporations.read_structures.v1"},
		SectionCorporationWalletBalances:      {"esi-wallet.read_corporation_wallets.v1"},
		SectionCorporationWalletJournal1:      journal,
		SectionCorporationWalletJournal2:      journal,
		SectionCorporationWalletJournal3:      journal,
		SectionCorporationWalletJournal4:      journal,
		SectionCorporationWalletJournal5:      journal,
		SectionCorporationWalletJournal6:      journal,
		SectionCorporationWalletJournal7:      journal,
		SectionCorporationWalletTransactions1: transactions,
		SectionCorporationWalletTransactions2: transactions,
		SectionCorporationWalletTransactions3: transactions,
		SectionCorporationWalletTransactions4: transactions,
		SectionCorporationWalletTransactions5: transactions,
		SectionCorporationWalletTransactions6: transactions,
		SectionCorporationWalletTransactions7: transactions,
	}
	scopes, ok := m[cs]
	if !ok {
		slog.Warn("Requested scopes for unknown section. Using default.", "section", cs)
		return set.Set[string]{}
	}
	return set.Of(scopes...)
}

// GeneralSection represents a general topic that can be updated, e.g. market prices
type GeneralSection string

var _ section = (*GeneralSection)(nil)

const (
	SectionEveCharacters   GeneralSection = "characters"    // character
	SectionEveCorporations GeneralSection = "corporations"  // corporation
	SectionEveEntities     GeneralSection = "entities"      // static-data
	SectionEveMarketPrices GeneralSection = "market_prices" // market
	SectionEveTypes        GeneralSection = "types"         // static-data
)

var GeneralSections = []GeneralSection{
	SectionEveCharacters,
	SectionEveCorporations,
	SectionEveEntities,
	SectionEveMarketPrices,
	SectionEveTypes,
}

var generalSectionTimeouts = map[GeneralSection]time.Duration{
	SectionEveCharacters:   1 * time.Hour,
	SectionEveCorporations: 1 * time.Hour,
	SectionEveEntities:     6 * time.Hour,
	SectionEveMarketPrices: 6 * time.Hour,
	SectionEveTypes:        24 * time.Hour,
}

func (gs GeneralSection) DisplayName() string {
	return makeSectionDisplayName(gs)
}

func (gs GeneralSection) IsSkippingChangeDetection() bool {
	return false
}

func (gs GeneralSection) Scopes() set.Set[string] {
	return set.Set[string]{}
}

func (gs GeneralSection) String() string {
	return string(gs)
}

func (gs GeneralSection) Timeout() time.Duration {
	duration, ok := generalSectionTimeouts[gs]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", gs)
		return generalSectionDefaultTimeout
	}
	return duration
}

// Scopes returns all required ESI scopes.
func Scopes() set.Set[string] {
	scopes := set.Of(
		"esi-characters.read_contacts.v1", // already requested and for planned feature
		"esi-mail.send_mail.v1",           // required for sending mail
		"esi-search.search_structures.v1", // required for new eden search
	)
	for _, s := range CharacterSections {
		scopes.AddSeq(s.Scopes().All())
	}
	for _, s := range CorporationSections {
		scopes.AddSeq(s.Scopes().All())
	}
	return scopes
}

type GeneralSectionUpdateParams struct {
	ForceUpdate       bool
	OnUpdateStarted   func()
	OnUpdateCompleted func()
	Section           GeneralSection
}

type CharacterSectionUpdateParams struct {
	CharacterID           int32
	ForceUpdate           bool
	MarketOrderRetention  time.Duration
	MaxMails              int
	MaxWalletTransactions int
	OnUpdateCompleted     func()
	OnUpdateStarted       func()
	Section               CharacterSection
}

type CorporationSectionUpdateParams struct {
	CorporationID         int32
	ForceUpdate           bool
	MaxWalletTransactions int
	OnUpdateStarted       func()
	OnUpdateCompleted     func()
	Section               CorporationSection
}
