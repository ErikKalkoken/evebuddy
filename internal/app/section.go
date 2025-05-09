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

const (
	corporationSectionDefaultTimeout = 3600 * time.Second
)

type CorporationSection string

// Updated corporation sections
const (
	SectionCorporationIndustryJobs CorporationSection = "industry_jobs"
)

var CorporationSections = []CorporationSection{
	SectionCorporationIndustryJobs,
}

func (cs CorporationSection) DisplayName() string {
	t := strings.ReplaceAll(string(cs), "_", " ")
	c := cases.Title(language.English)
	t = c.String(t)
	return t
}

// Timeout returns the time until the data of an update section becomes stale.
func (cs CorporationSection) Timeout() time.Duration {
	m := map[CorporationSection]time.Duration{
		SectionCorporationIndustryJobs: 300 * time.Second,
	}
	duration, ok := m[cs]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", cs)
		return corporationSectionDefaultTimeout
	}
	return duration
}

// Role returns the required role for fetching data for a section from ESI.
func (cs CorporationSection) Role() Role {
	m := map[CorporationSection]Role{
		SectionCorporationIndustryJobs: RoleFactoryManager,
	}
	role, ok := m[cs]
	if !ok {
		slog.Warn("Requested role for unknown section. Using default.", "section", cs)
		return RoleDirector
	}
	return role
}

type CorporationUpdateSectionParams struct {
	CorporationID int32
	Section       CorporationSection
	ForceUpdate   bool
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

// A general section represents a topic that can be updated, e.g. market prices
type GeneralSection string

const (
	SectionEveCharacters   GeneralSection = "characters"
	SectionEveCorporations GeneralSection = "corporations"
	SectionEveMarketPrices GeneralSection = "market_prices"
	SectionEveTypes        GeneralSection = "types"
)

var GeneralSections = []GeneralSection{
	SectionEveCharacters,
	SectionEveCorporations,
	SectionEveMarketPrices,
	SectionEveTypes,
}

var generalSectionTimeouts = map[GeneralSection]time.Duration{
	SectionEveCharacters:   4 * time.Hour,
	SectionEveCorporations: 4 * time.Hour,
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

// Updates status of a general section
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
