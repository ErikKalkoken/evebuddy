package app

import (
	"log/slog"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ErikKalkoken/evebuddy/internal/set"
)

const (
	characterSectionDefaultTimeout   = 3600 * time.Second
	corporationSectionDefaultTimeout = 3600 * time.Second
	generalSectionDefaultTimeout     = 24 * time.Hour
	sectionErrorTimeout              = 120 * time.Second
)

type section interface {
	DisplayName() string
	Scopes() set.Set[string]
	String() string
	Timeout() time.Duration
}

func makeSectionDisplayName(cs section) string {
	t := strings.ReplaceAll(cs.String(), "_", " ")
	c := cases.Title(language.English)
	t = c.String(t)
	return t
}

type CharacterSection string

var _ section = (*CharacterSection)(nil)

// Updated character sections
const (
	SectionCharacterAssets             CharacterSection = "assets"
	SectionCharacterAttributes         CharacterSection = "attributes"
	SectionCharacterContracts          CharacterSection = "contracts"
	SectionCharacterImplants           CharacterSection = "implants"
	SectionCharacterIndustryJobs       CharacterSection = "industry_jobs"
	SectionCharacterJumpClones         CharacterSection = "jump_clones"
	SectionCharacterLocation           CharacterSection = "location"
	SectionCharacterMailLabels         CharacterSection = "mail_labels"
	SectionCharacterMailLists          CharacterSection = "mail_lists"
	SectionCharacterMails              CharacterSection = "mails"
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
	SectionCharacterContracts,
	SectionCharacterImplants,
	SectionCharacterIndustryJobs,
	SectionCharacterJumpClones,
	SectionCharacterLocation,
	SectionCharacterMailLabels,
	SectionCharacterMailLists,
	SectionCharacterMails,
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

func (cs CharacterSection) String() string {
	return string(cs)
}

// Timeout returns the time until the data of an update section becomes stale.
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
		SectionCharacterMails:              60 * time.Second, // minimum 30 seconds
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

// Scopes returns the required scopes for fetching data for a section from ESI.
func (cs CharacterSection) Scopes() set.Set[string] {
	m := map[CharacterSection][]string{
		SectionCharacterAssets:             {"esi-assets.read_assets.v1", "esi-universe.read_structures.v1"},
		SectionCharacterAttributes:         {"esi-skills.read_skills.v1"},
		SectionCharacterContracts:          {"esi-contracts.read_character_contracts.v1", "esi-universe.read_structures.v1"},
		SectionCharacterImplants:           {"esi-clones.read_implants.v1"},
		SectionCharacterIndustryJobs:       {"esi-industry.read_character_jobs.v1", "esi-universe.read_structures.v1"},
		SectionCharacterJumpClones:         {"esi-clones.read_clones.v1", "esi-universe.read_structures.v1"},
		SectionCharacterLocation:           {"esi-location.read_location.v1", "esi-universe.read_structures.v1"},
		SectionCharacterMailLabels:         {"esi-mail.read_mail.v1"},
		SectionCharacterMailLists:          {"esi-mail.read_mail.v1"},
		SectionCharacterMails:              {"esi-mail.organize_mail.v1", "esi-mail.read_mail.v1"},
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

type CorporationSection string

var _ section = (*CorporationSection)(nil)

// Updated corporation sections
const (
	SectionCorporationDivisions           CorporationSection = "divisions"
	SectionCorporationIndustryJobs        CorporationSection = "industry_jobs"
	SectionCorporationMembers             CorporationSection = "members"
	SectionCorporationWalletBalances      CorporationSection = "wallet_balances"
	SectionCorporationWalletJournal1      CorporationSection = "wallet_journal_1"
	SectionCorporationWalletJournal2      CorporationSection = "wallet_journal_2"
	SectionCorporationWalletJournal3      CorporationSection = "wallet_journal_3"
	SectionCorporationWalletJournal4      CorporationSection = "wallet_journal_4"
	SectionCorporationWalletJournal5      CorporationSection = "wallet_journal_5"
	SectionCorporationWalletJournal6      CorporationSection = "wallet_journal_6"
	SectionCorporationWalletJournal7      CorporationSection = "wallet_journal_7"
	SectionCorporationWalletTransactions1 CorporationSection = "wallet_transactions_1"
	SectionCorporationWalletTransactions2 CorporationSection = "wallet_transactions_2"
	SectionCorporationWalletTransactions3 CorporationSection = "wallet_transactions_3"
	SectionCorporationWalletTransactions4 CorporationSection = "wallet_transactions_4"
	SectionCorporationWalletTransactions5 CorporationSection = "wallet_transactions_5"
	SectionCorporationWalletTransactions6 CorporationSection = "wallet_transactions_6"
	SectionCorporationWalletTransactions7 CorporationSection = "wallet_transactions_7"
)

var CorporationSections = []CorporationSection{
	SectionCorporationDivisions,
	SectionCorporationIndustryJobs,
	SectionCorporationMembers,
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

// Timeout returns the time until the data of an update section becomes stale.
func (cs CorporationSection) Timeout() time.Duration {
	const (
		walletTransactions = 3600 * time.Second
		walletJournal      = 3600 * time.Second
	)
	m := map[CorporationSection]time.Duration{
		SectionCorporationDivisions:           3600 * time.Second,
		SectionCorporationIndustryJobs:        300 * time.Second,
		SectionCorporationMembers:             3600 * time.Second,
		SectionCorporationWalletBalances:      300 * time.Second,
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
		SectionCorporationDivisions:           {RoleDirector},
		SectionCorporationIndustryJobs:        {RoleFactoryManager},
		SectionCorporationMembers:             {},
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

// Scopes returns the required scopes for fetching data for a section from ESI.
func (cs CorporationSection) Scopes() set.Set[string] {
	journal := []string{"esi-wallet.read_corporation_wallets.v1"}
	transactions := []string{"esi-wallet.read_corporation_wallets.v1", "esi-universe.read_structures.v1"}
	m := map[CorporationSection][]string{
		SectionCorporationDivisions:           {"esi-corporations.read_divisions.v1"},
		SectionCorporationIndustryJobs:        {"esi-industry.read_corporation_jobs.v1"},
		SectionCorporationMembers:             {"esi-corporations.read_corporation_membership.v1"},
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

// GeneralSection represents a topic that can be updated, e.g. market prices
type GeneralSection string

var _ section = (*GeneralSection)(nil)

const (
	SectionEveCharacters   GeneralSection = "characters"
	SectionEveCorporations GeneralSection = "corporations"
	SectionEveEntities     GeneralSection = "entities"
	SectionEveMarketPrices GeneralSection = "market_prices"
	SectionEveTypes        GeneralSection = "types"
)

var GeneralSections = []GeneralSection{
	SectionEveCharacters,
	SectionEveCorporations,
	SectionEveEntities,
	SectionEveMarketPrices,
	SectionEveTypes,
}

var generalSectionTimeouts = map[GeneralSection]time.Duration{
	SectionEveCharacters:   4 * time.Hour,
	SectionEveCorporations: 4 * time.Hour,
	SectionEveEntities:     24 * time.Hour,
	SectionEveMarketPrices: 6 * time.Hour,
	SectionEveTypes:        24 * time.Hour,
}

func (gs GeneralSection) DisplayName() string {
	return makeSectionDisplayName(gs)
}

// Timeout returns the time until the data of an update section becomes stale.
func (gs GeneralSection) Timeout() time.Duration {
	duration, ok := generalSectionTimeouts[gs]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", gs)
		return generalSectionDefaultTimeout
	}
	return duration
}

func (gs GeneralSection) Scopes() set.Set[string] {
	return set.Set[string]{}
}

func (gs GeneralSection) String() string {
	return string(gs)
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

type GeneralUpdateSectionParams struct {
	ForceUpdate       bool
	OnUpdateStarted   func()
	OnUpdateCompleted func()
	Section           GeneralSection
}

type CharacterUpdateSectionParams struct {
	CharacterID           int32
	ForceUpdate           bool
	MaxMails              int
	MaxWalletTransactions int
	OnUpdateStarted       func()
	OnUpdateCompleted     func()
	Section               CharacterSection
}

type CorporationUpdateSectionParams struct {
	CorporationID         int32
	ForceUpdate           bool
	MaxWalletTransactions int
	OnUpdateStarted       func()
	OnUpdateCompleted     func()
	Section               CorporationSection
}
