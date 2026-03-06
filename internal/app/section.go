package app

import (
	"log/slog"
	"strings"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

const (
	characterSectionDefaultTimeout   = 3600 * time.Second
	corporationSectionDefaultTimeout = 3600 * time.Second
	eveUniverseSectionDefaultTimeout = 24 * time.Hour
	sectionErrorTimeout              = 120 * time.Second
)

// section defines the interface for all section types.
type section interface {
	// DisplayName returns the output friendly name of a section.
	DisplayName() string

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
	SectionCharacterAssets             CharacterSection = "assets"
	SectionCharacterAttributes         CharacterSection = "attributes"
	SectionCharacterContacts           CharacterSection = "contacts"
	SectionCharacterContactLabels      CharacterSection = "contact_labels"
	SectionCharacterContracts          CharacterSection = "contracts"
	SectionCharacterImplants           CharacterSection = "implants"
	SectionCharacterIndustryJobs       CharacterSection = "industry_jobs"
	SectionCharacterJumpClones         CharacterSection = "jump_clones"
	SectionCharacterLocation           CharacterSection = "location"
	SectionCharacterLoyaltyPoints      CharacterSection = "loyalty_points"
	SectionCharacterMailHeaders        CharacterSection = "mail_headers"
	SectionCharacterMailLabels         CharacterSection = "mail_labels"
	SectionCharacterMailLists          CharacterSection = "mail_lists"
	SectionCharacterMarketOrders       CharacterSection = "market_orders"
	SectionCharacterNotifications      CharacterSection = "notifications"
	SectionCharacterOnline             CharacterSection = "online"
	SectionCharacterPlanets            CharacterSection = "planets"
	SectionCharacterRoles              CharacterSection = "roles"
	SectionCharacterShip               CharacterSection = "ship"
	SectionCharacterSkillqueue         CharacterSection = "skillqueue"
	SectionCharacterSkills             CharacterSection = "skills"
	SectionCharacterWalletBalance      CharacterSection = "wallet_balance"
	SectionCharacterWalletJournal      CharacterSection = "wallet_journal"
	SectionCharacterWalletTransactions CharacterSection = "wallet_transactions"
)

var CharacterSections = []CharacterSection{
	SectionCharacterAssets,
	SectionCharacterAttributes,
	SectionCharacterContacts,
	SectionCharacterContactLabels,
	SectionCharacterContracts,
	SectionCharacterImplants,
	SectionCharacterIndustryJobs,
	SectionCharacterJumpClones,
	SectionCharacterLocation,
	SectionCharacterLoyaltyPoints,
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

func (cs CharacterSection) Scopes() set.Set[string] {
	m := map[CharacterSection][]string{
		SectionCharacterAssets:             {goesi.ScopeAssetsReadAssetsV1, goesi.ScopeUniverseReadStructuresV1},
		SectionCharacterAttributes:         {goesi.ScopeSkillsReadSkillsV1},
		SectionCharacterContacts:           {goesi.ScopeCharactersReadContactsV1},
		SectionCharacterContactLabels:      {goesi.ScopeCharactersReadContactsV1},
		SectionCharacterContracts:          {goesi.ScopeContractsReadCharacterContractsV1, goesi.ScopeUniverseReadStructuresV1},
		SectionCharacterImplants:           {goesi.ScopeClonesReadImplantsV1},
		SectionCharacterIndustryJobs:       {goesi.ScopeIndustryReadCharacterJobsV1, goesi.ScopeUniverseReadStructuresV1},
		SectionCharacterJumpClones:         {goesi.ScopeClonesReadClonesV1, goesi.ScopeUniverseReadStructuresV1},
		SectionCharacterLocation:           {goesi.ScopeLocationReadLocationV1, goesi.ScopeUniverseReadStructuresV1},
		SectionCharacterLoyaltyPoints:      {goesi.ScopeCharactersReadLoyaltyV1},
		SectionCharacterMailHeaders:        {goesi.ScopeMailOrganizeMailV1, goesi.ScopeMailReadMailV1},
		SectionCharacterMailLabels:         {goesi.ScopeMailReadMailV1},
		SectionCharacterMailLists:          {goesi.ScopeMailReadMailV1},
		SectionCharacterMarketOrders:       {goesi.ScopeMarketsReadCharacterOrdersV1},
		SectionCharacterNotifications:      {goesi.ScopeCharactersReadNotificationsV1, goesi.ScopeUniverseReadStructuresV1},
		SectionCharacterOnline:             {goesi.ScopeLocationReadOnlineV1},
		SectionCharacterPlanets:            {goesi.ScopePlanetsManagePlanetsV1},
		SectionCharacterRoles:              {goesi.ScopeCharactersReadCorporationRolesV1},
		SectionCharacterShip:               {goesi.ScopeLocationReadShipTypeV1},
		SectionCharacterSkillqueue:         {goesi.ScopeSkillsReadSkillqueueV1},
		SectionCharacterSkills:             {goesi.ScopeSkillsReadSkillsV1},
		SectionCharacterWalletBalance:      {goesi.ScopeWalletReadCharacterWalletV1},
		SectionCharacterWalletJournal:      {goesi.ScopeWalletReadCharacterWalletV1},
		SectionCharacterWalletTransactions: {goesi.ScopeWalletReadCharacterWalletV1, goesi.ScopeUniverseReadStructuresV1},
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
		SectionCharacterContacts:           300 * time.Second,
		SectionCharacterContactLabels:      300 * time.Second,
		SectionCharacterContracts:          300 * time.Second,
		SectionCharacterImplants:           120 * time.Second,
		SectionCharacterIndustryJobs:       300 * time.Second,
		SectionCharacterJumpClones:         120 * time.Second,
		SectionCharacterLocation:           300 * time.Second, // minimum 5 seconds
		SectionCharacterLoyaltyPoints:      3600 * time.Second,
		SectionCharacterMailHeaders:        60 * time.Second, // minimum 30 seconds
		SectionCharacterMailLabels:         60 * time.Second, // minimum 30 seconds
		SectionCharacterMailLists:          120 * time.Second,
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
	journal := []string{goesi.ScopeWalletReadCorporationWalletsV1}
	transactions := []string{goesi.ScopeWalletReadCorporationWalletsV1, goesi.ScopeUniverseReadStructuresV1}
	m := map[CorporationSection][]string{
		SectionCorporationAssets:              {goesi.ScopeAssetsReadCorporationAssetsV1},
		SectionCorporationContracts:           {goesi.ScopeContractsReadCorporationContractsV1},
		SectionCorporationDivisions:           {goesi.ScopeCorporationsReadDivisionsV1},
		SectionCorporationIndustryJobs:        {goesi.ScopeIndustryReadCorporationJobsV1},
		SectionCorporationMembers:             {goesi.ScopeCorporationsReadCorporationMembershipV1},
		SectionCorporationStructures:          {goesi.ScopeCorporationsReadStructuresV1},
		SectionCorporationWalletBalances:      {goesi.ScopeWalletReadCorporationWalletsV1},
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

// EveUniverseSection represents a topic from EveUniverse that can be updated, e.g. market prices
type EveUniverseSection string

var _ section = (*EveUniverseSection)(nil)

const (
	SectionEveCharacters   EveUniverseSection = "characters"    // character
	SectionEveCorporations EveUniverseSection = "corporations"  // corporation
	SectionEveEntities     EveUniverseSection = "entities"      // static-data
	SectionEveMarketPrices EveUniverseSection = "market_prices" // market
	SectionEveTypes        EveUniverseSection = "types"         // static-data
)

var EveUniverseSections = []EveUniverseSection{
	SectionEveCharacters,
	SectionEveCorporations,
	SectionEveEntities,
	SectionEveMarketPrices,
	SectionEveTypes,
}

var eveUniverseSectionTimeouts = map[EveUniverseSection]time.Duration{
	SectionEveCharacters:   1 * time.Hour,
	SectionEveCorporations: 1 * time.Hour,
	SectionEveEntities:     6 * time.Hour,
	SectionEveMarketPrices: 6 * time.Hour,
	SectionEveTypes:        24 * time.Hour,
}

func (gs EveUniverseSection) DisplayName() string {
	return makeSectionDisplayName(gs)
}

func (gs EveUniverseSection) Scopes() set.Set[string] {
	return set.Set[string]{}
}

func (gs EveUniverseSection) String() string {
	return string(gs)
}

func (gs EveUniverseSection) Timeout() time.Duration {
	duration, ok := eveUniverseSectionTimeouts[gs]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", gs)
		return eveUniverseSectionDefaultTimeout
	}
	return duration
}

// Scopes returns all required ESI scopes.
func Scopes() set.Set[string] {
	scopes := set.Of(
		goesi.ScopeCharactersReadContactsV1, // already requested and for planned feature
		goesi.ScopeMailSendMailV1,           // required for sending mail
		goesi.ScopeSearchSearchStructuresV1, // required for new eden search
	)
	for _, s := range CharacterSections {
		scopes.AddSeq(s.Scopes().All())
	}
	for _, s := range CorporationSections {
		scopes.AddSeq(s.Scopes().All())
	}
	return scopes
}

type EveUniverseSectionUpdateParams struct {
	ForceUpdate bool
	Section     EveUniverseSection
}

type CharacterSectionUpdateParams struct {
	CharacterID int64
	ForceUpdate bool
	Section     CharacterSection
}

type CorporationSectionUpdateParams struct {
	CorporationID int64
	ForceUpdate   bool
	Section       CorporationSection
}
