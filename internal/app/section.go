package app

import (
	"log/slog"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/set"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	characterSectionDefaultTimeout = 3600 * time.Second
)

type CharacterSection string

// Updated character sections
const (
	SectionAssets             CharacterSection = "assets"
	SectionAttributes         CharacterSection = "attributes"
	SectionContracts          CharacterSection = "contracts"
	SectionImplants           CharacterSection = "implants"
	SectionIndustryJobs       CharacterSection = "industry_jobs"
	SectionJumpClones         CharacterSection = "jump_clones"
	SectionLocation           CharacterSection = "location"
	SectionMailLabels         CharacterSection = "mail_labels"
	SectionMailLists          CharacterSection = "mail_lists"
	SectionMails              CharacterSection = "mails"
	SectionNotifications      CharacterSection = "notifications"
	SectionOnline             CharacterSection = "online"
	SectionPlanets            CharacterSection = "planets"
	SectionRoles              CharacterSection = "roles"
	SectionShip               CharacterSection = "ship"
	SectionSkillqueue         CharacterSection = "skillqueue"
	SectionSkills             CharacterSection = "skills"
	SectionWalletBalance      CharacterSection = "wallet_balance"
	SectionWalletJournal      CharacterSection = "wallet_journal"
	SectionWalletTransactions CharacterSection = "wallet_transactions"
)

var CharacterSections = []CharacterSection{
	SectionAssets,
	SectionAttributes,
	SectionContracts,
	SectionImplants,
	SectionIndustryJobs,
	SectionJumpClones,
	SectionLocation,
	SectionMailLabels,
	SectionMailLists,
	SectionMails,
	SectionNotifications,
	SectionOnline,
	SectionPlanets,
	SectionRoles,
	SectionShip,
	SectionSkillqueue,
	SectionSkills,
	SectionWalletBalance,
	SectionWalletJournal,
	SectionWalletTransactions,
}

func (cs CharacterSection) DisplayName() string {
	t := strings.ReplaceAll(string(cs), "_", " ")
	c := cases.Title(language.English)
	t = c.String(t)
	return t
}

// Timeout returns the time until the data of an update section becomes stale.
func (cs CharacterSection) Timeout() time.Duration {
	var m = map[CharacterSection]time.Duration{
		SectionAssets:             3600 * time.Second,
		SectionAttributes:         120 * time.Second,
		SectionContracts:          300 * time.Second,
		SectionImplants:           120 * time.Second,
		SectionIndustryJobs:       300 * time.Second,
		SectionJumpClones:         120 * time.Second,
		SectionLocation:           300 * time.Second, // minimum 5 seconds
		SectionMailLabels:         60 * time.Second,  // minimum 30 seconds
		SectionMailLists:          120 * time.Second,
		SectionMails:              60 * time.Second, // minimum 30 seconds
		SectionNotifications:      600 * time.Second,
		SectionOnline:             300 * time.Second, // minimum 30 seconds
		SectionPlanets:            600 * time.Second,
		SectionRoles:              3600 * time.Second,
		SectionShip:               300 * time.Second, // minimum 5 seconds
		SectionSkillqueue:         120 * time.Second,
		SectionSkills:             120 * time.Second,
		SectionWalletBalance:      120 * time.Second,
		SectionWalletJournal:      3600 * time.Second,
		SectionWalletTransactions: 3600 * time.Second,
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
		SectionAssets:             {"esi-assets.read_assets.v1", "esi-universe.read_structures.v1"},
		SectionAttributes:         {"esi-skills.read_skills.v1"},
		SectionContracts:          {"esi-contracts.read_character_contracts.v1", "esi-universe.read_structures.v1"},
		SectionImplants:           {"esi-clones.read_implants.v1"},
		SectionIndustryJobs:       {"esi-industry.read_character_jobs.v1", "esi-universe.read_structures.v1"},
		SectionJumpClones:         {"esi-clones.read_clones.v1", "esi-universe.read_structures.v1"},
		SectionLocation:           {"esi-location.read_location.v1", "esi-universe.read_structures.v1"},
		SectionMailLabels:         {"esi-mail.read_mail.v1"},
		SectionMailLists:          {"esi-mail.read_mail.v1"},
		SectionMails:              {"esi-mail.organize_mail.v1", "esi-mail.read_mail.v1"},
		SectionNotifications:      {"esi-characters.read_notifications.v1", "esi-universe.read_structures.v1"},
		SectionOnline:             {"esi-location.read_online.v1"},
		SectionPlanets:            {"esi-planets.manage_planets.v1"},
		SectionRoles:              {"esi-characters.read_corporation_roles.v1"},
		SectionShip:               {"esi-location.read_ship_type.v1"},
		SectionSkillqueue:         {"esi-skills.read_skillqueue.v1"},
		SectionSkills:             {"esi-skills.read_skills.v1"},
		SectionWalletBalance:      {"esi-wallet.read_character_wallet.v1"},
		SectionWalletJournal:      {"esi-wallet.read_character_wallet.v1"},
		SectionWalletTransactions: {"esi-wallet.read_character_wallet.v1", "esi-universe.read_structures.v1"},
	}
	scopes, ok := m[cs]
	if !ok {
		slog.Warn("Requested scopes for unknown section. Using default.", "section", cs)
		return set.Of[string]()
	}
	return set.Of(scopes...)
}

type CharacterUpdateSectionParams struct {
	CharacterID           int32
	Section               CharacterSection
	ForceUpdate           bool
	MaxMails              int
	MaxWalletTransactions int
}

type CharacterSectionStatus struct {
	CharacterID   int32
	CharacterName string
	CompletedAt   time.Time
	ContentHash   string
	ErrorMessage  string
	Section       CharacterSection
	StartedAt     time.Time
	UpdatedAt     time.Time
}

func (s CharacterSectionStatus) HasError() bool {
	return s.ErrorMessage != ""
}

func (s CharacterSectionStatus) IsExpired() bool {
	if s.CompletedAt.IsZero() {
		return true
	}
	timeout := s.Section.Timeout()
	deadline := s.CompletedAt.Add(timeout)
	return time.Now().After(deadline)
}

func (s CharacterSectionStatus) IsMissing() bool {
	return s.CompletedAt.IsZero()
}

const (
	corporationSectionDefaultTimeout = 3600 * time.Second
)

type CorporationSection string

// Updated corporation sections
const (
	SectionCorporationDivisions           CorporationSection = "divisions"
	SectionCorporationIndustryJobs        CorporationSection = "industry_jobs"
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

func (cs CorporationSection) DisplayName() string {
	t := strings.ReplaceAll(string(cs), "_", " ")
	c := cases.Title(language.English)
	t = c.String(t)
	return t
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

// Timeout returns the time until the data of an update section becomes stale.
func (cs CorporationSection) Timeout() time.Duration {
	const (
		walletTransactions = 3600 * time.Second
		walletJournal      = 3600 * time.Second
	)
	m := map[CorporationSection]time.Duration{
		SectionCorporationIndustryJobs:        300 * time.Second,
		SectionCorporationWalletBalances:      300 * time.Second,
		SectionCorporationWalletJournal1:      walletJournal,
		SectionCorporationWalletJournal2:      walletJournal,
		SectionCorporationWalletJournal3:      walletJournal,
		SectionCorporationWalletJournal4:      walletJournal,
		SectionCorporationWalletJournal5:      walletJournal,
		SectionCorporationWalletJournal6:      walletJournal,
		SectionCorporationWalletJournal7:      walletJournal,
		SectionCorporationDivisions:           3600 * time.Second,
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

// TODO: Extend to multiple roles per sections

// Role returns the required role for fetching data for a section from ESI.
func (cs CorporationSection) Role() Role {
	const (
		walletTransactions = RoleAccountant
		walletJournal      = RoleAccountant
	)
	m := map[CorporationSection]Role{
		SectionCorporationIndustryJobs:        RoleFactoryManager,
		SectionCorporationWalletBalances:      RoleAccountant,
		SectionCorporationWalletJournal1:      walletJournal,
		SectionCorporationWalletJournal2:      walletJournal,
		SectionCorporationWalletJournal3:      walletJournal,
		SectionCorporationWalletJournal4:      walletJournal,
		SectionCorporationWalletJournal5:      walletJournal,
		SectionCorporationWalletJournal6:      walletJournal,
		SectionCorporationWalletJournal7:      walletJournal,
		SectionCorporationDivisions:           RoleDirector,
		SectionCorporationWalletTransactions1: walletTransactions,
		SectionCorporationWalletTransactions2: walletTransactions,
		SectionCorporationWalletTransactions3: walletTransactions,
		SectionCorporationWalletTransactions4: walletTransactions,
		SectionCorporationWalletTransactions5: walletTransactions,
		SectionCorporationWalletTransactions6: walletTransactions,
		SectionCorporationWalletTransactions7: walletTransactions,
	}
	role, ok := m[cs]
	if !ok {
		slog.Warn("Requested role for unknown section. Using default.", "section", cs)
		return RoleDirector
	}
	return role
}

// Scopes returns the required scopes for fetching data for a section from ESI.
func (cs CorporationSection) Scopes() set.Set[string] {
	journal := []string{"esi-wallet.read_corporation_wallets.v1"}
	transactions := []string{"esi-wallet.read_corporation_wallets.v1", "esi-universe.read_structures.v1"}
	m := map[CorporationSection][]string{
		SectionCorporationIndustryJobs:        {"esi-industry.read_corporation_jobs.v1"},
		SectionCorporationWalletBalances:      {"esi-wallet.read_corporation_wallets.v1"},
		SectionCorporationWalletJournal1:      journal,
		SectionCorporationWalletJournal2:      journal,
		SectionCorporationWalletJournal3:      journal,
		SectionCorporationWalletJournal4:      journal,
		SectionCorporationWalletJournal5:      journal,
		SectionCorporationWalletJournal6:      journal,
		SectionCorporationWalletJournal7:      journal,
		SectionCorporationDivisions:           {"esi-corporations.read_divisions.v1"},
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

type CorporationUpdateSectionParams struct {
	CorporationID         int32
	ForceUpdate           bool
	MaxWalletTransactions int
	Section               CorporationSection
}

type CorporationSectionStatus struct {
	Comment         string
	CorporationID   int32
	CorporationName string
	CompletedAt     time.Time
	ContentHash     string
	ErrorMessage    string
	Section         CorporationSection
	StartedAt       time.Time
	UpdatedAt       time.Time
}

func (s CorporationSectionStatus) HasError() bool {
	return s.ErrorMessage != ""
}

func (s CorporationSectionStatus) IsExpired() bool {
	if s.CompletedAt.IsZero() {
		return true
	}
	timeout := s.Section.Timeout()
	deadline := s.CompletedAt.Add(timeout)
	return time.Now().After(deadline)
}

const (
	generalSectionDefaultTimeout = 24 * time.Hour
)

// GeneralSection represents a topic that can be updated, e.g. market prices
type GeneralSection string

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
	t := strings.ReplaceAll(string(gs), "_", " ")
	c := cases.Title(language.English)
	t = c.String(t)
	return t
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

// GeneralSectionStatus represents the status of a general section.
type GeneralSectionStatus struct {
	ContentHash  string
	ErrorMessage string
	CompletedAt  time.Time
	Section      GeneralSection
	StartedAt    time.Time
	UpdatedAt    time.Time
}

func (s GeneralSectionStatus) HasError() bool {
	return s.ErrorMessage != ""
}

func (s GeneralSectionStatus) IsExpired() bool {
	if s.CompletedAt.IsZero() {
		return true
	}
	timeout := s.Section.Timeout()
	deadline := s.CompletedAt.Add(timeout)
	return time.Now().After(deadline)
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
