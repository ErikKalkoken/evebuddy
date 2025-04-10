package app

import (
	"log/slog"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const characterSectionDefaultTimeout = 3600 * time.Second

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
