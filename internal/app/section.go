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
	generalSectionDefaultTimeout   = 24 * time.Hour
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
	SectionRoles,
	SectionPlanets,
	SectionShip,
	SectionSkillqueue,
	SectionSkills,
	SectionWalletBalance,
	SectionWalletJournal,
	SectionWalletTransactions,
}

var characterSectionTimeouts = map[CharacterSection]time.Duration{
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

func (cs CharacterSection) DisplayName() string {
	t := strings.ReplaceAll(string(cs), "_", " ")
	c := cases.Title(language.English)
	t = c.String(t)
	return t
}

// Timeout returns the time until the data of an update section becomes stale.
func (cs CharacterSection) Timeout() time.Duration {
	duration, ok := characterSectionTimeouts[cs]
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
	ID            int64
	CharacterID   int32
	CharacterName string
	CompletedAt   time.Time
	ContentHash   string
	ErrorMessage  string
	Section       CharacterSection
	StartedAt     time.Time
	UpdatedAt     time.Time
}

func (s CharacterSectionStatus) IsOK() bool {
	return s.ErrorMessage == ""
}

func (s CharacterSectionStatus) IsExpired() bool {
	if s.CompletedAt.IsZero() {
		return true
	}
	timeout := s.Section.Timeout()
	deadline := s.CompletedAt.Add(timeout)
	return time.Now().After(deadline)
}

// A general section represents a topic that can be updated, e.g. market prices
type GeneralSection string

const (
	SectionEveCategories   GeneralSection = "Eve_Categories"
	SectionEveCharacters   GeneralSection = "Eve_Characters"
	SectionEveMarketPrices GeneralSection = "Eve_MarketPrices"
)

var GeneralSections = []GeneralSection{
	SectionEveCategories,
	SectionEveCharacters,
	SectionEveMarketPrices,
}

var generalSectionTimeouts = map[GeneralSection]time.Duration{
	SectionEveCategories:   24 * time.Hour,
	SectionEveCharacters:   1 * time.Hour,
	SectionEveMarketPrices: 6 * time.Hour,
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
	ID           int64
	ContentHash  string
	ErrorMessage string
	CompletedAt  time.Time
	Section      GeneralSection
	StartedAt    time.Time
	UpdatedAt    time.Time
}

func (s GeneralSectionStatus) IsOK() bool {
	return s.ErrorMessage == ""
}

func (s GeneralSectionStatus) IsExpired() bool {
	if s.CompletedAt.IsZero() {
		return true
	}
	timeout := s.Section.Timeout()
	deadline := s.CompletedAt.Add(timeout)
	return time.Now().After(deadline)
}
