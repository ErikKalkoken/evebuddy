package app

import (
	"log/slog"
	"strings"
	"time"

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
	m := map[CorporationSection]time.Duration{
		SectionCorporationIndustryJobs:        300 * time.Second,
		SectionCorporationWalletBalances:      300 * time.Second,
		SectionCorporationWalletJournal1:      3600 * time.Second,
		SectionCorporationWalletJournal2:      3600 * time.Second,
		SectionCorporationWalletJournal3:      3600 * time.Second,
		SectionCorporationWalletJournal4:      3600 * time.Second,
		SectionCorporationWalletJournal5:      3600 * time.Second,
		SectionCorporationWalletJournal6:      3600 * time.Second,
		SectionCorporationWalletJournal7:      3600 * time.Second,
		SectionCorporationDivisions:           3600 * time.Second,
		SectionCorporationWalletTransactions1: 3600 * time.Second,
		SectionCorporationWalletTransactions2: 3600 * time.Second,
		SectionCorporationWalletTransactions3: 3600 * time.Second,
		SectionCorporationWalletTransactions4: 3600 * time.Second,
		SectionCorporationWalletTransactions5: 3600 * time.Second,
		SectionCorporationWalletTransactions6: 3600 * time.Second,
		SectionCorporationWalletTransactions7: 3600 * time.Second,
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
	m := map[CorporationSection]Role{
		SectionCorporationIndustryJobs:        RoleFactoryManager,
		SectionCorporationWalletBalances:      RoleAccountant,
		SectionCorporationWalletJournal1:      RoleAccountant,
		SectionCorporationDivisions:           RoleDirector,
		SectionCorporationWalletTransactions1: RoleAccountant,
	}
	role, ok := m[cs]
	if !ok {
		slog.Warn("Requested role for unknown section. Using default.", "section", cs)
		return RoleDirector
	}
	return role
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
