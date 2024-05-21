// Package model contains the entity objects, which are used across the app.
package model

import (
	"database/sql"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const defaultUpdateSectionTimeout = 3600 * time.Second

// An Eve Online character owners by the user.
type Character struct {
	EveCharacter  *EveCharacter
	Home          *Location
	ID            int32
	LastLoginAt   sql.NullTime
	Location      *Location
	Ship          *EveType
	TotalSP       sql.NullInt64
	UnallocatedSP sql.NullInt64
	WalletBalance sql.NullFloat64
}

// A shortened version of Character.
type CharacterShort struct {
	ID              int32
	Name            string
	CorporationName string
}

type CharacterSection string

func (s CharacterSection) Name() string {
	t := strings.ReplaceAll(string(s), "_", " ")
	c := cases.Title(language.English)
	t = c.String(t)
	return t
}

// Updated character sections
const (
	CharacterSectionHome               CharacterSection = "home"
	CharacterSectionImplants           CharacterSection = "implants"
	CharacterSectionLocation           CharacterSection = "location"
	CharacterSectionMailLists          CharacterSection = "mail_lists"
	CharacterSectionMailLabels         CharacterSection = "mail_labels"
	CharacterSectionMails              CharacterSection = "mails"
	CharacterSectionOnline             CharacterSection = "online"
	CharacterSectionShip               CharacterSection = "ship"
	CharacterSectionSkills             CharacterSection = "skills"
	CharacterSectionSkillqueue         CharacterSection = "skillqueue"
	CharacterSectionWalletBalance      CharacterSection = "wallet_balance"
	CharacterSectionWalletJournal      CharacterSection = "wallet_journal"
	CharacterSectionWalletTransactions CharacterSection = "wallet_transactions"
)

var CharacterSections = []CharacterSection{
	CharacterSectionHome,
	CharacterSectionImplants,
	CharacterSectionLocation,
	CharacterSectionMailLabels,
	CharacterSectionMailLists,
	CharacterSectionMails,
	CharacterSectionOnline,
	CharacterSectionShip,
	CharacterSectionSkills,
	CharacterSectionSkillqueue,
	CharacterSectionWalletBalance,
	CharacterSectionWalletJournal,
	CharacterSectionWalletTransactions,
}

// Timeout returns the time until the data of an update section becomes stale.
func (section CharacterSection) Timeout() time.Duration {
	m := map[CharacterSection]time.Duration{
		CharacterSectionHome:               120 * time.Second,
		CharacterSectionImplants:           120 * time.Second,
		CharacterSectionLocation:           30 * time.Second, // 5 seconds min
		CharacterSectionMailLabels:         30 * time.Second,
		CharacterSectionMailLists:          120 * time.Second,
		CharacterSectionMails:              30 * time.Second,
		CharacterSectionOnline:             60 * time.Second,
		CharacterSectionShip:               30 * time.Second, // 5 seconds min
		CharacterSectionSkillqueue:         120 * time.Second,
		CharacterSectionSkills:             120 * time.Second,
		CharacterSectionWalletBalance:      120 * time.Second,
		CharacterSectionWalletJournal:      3600 * time.Second,
		CharacterSectionWalletTransactions: 3600 * time.Second,
	}
	duration, ok := m[section]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", section)
		duration = defaultUpdateSectionTimeout
	}
	return duration
}

type CharacterUpdateStatus struct {
	ID            int64
	CharacterID   int32
	ErrorMessage  string
	Section       CharacterSection
	LastUpdatedAt sql.NullTime
	ContentHash   string
}

func (s CharacterUpdateStatus) IsOK() bool {
	return s.ErrorMessage == ""
}
