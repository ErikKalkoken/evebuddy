// Package model contains the entity objects, which are used across the app.
package model

import (
	"database/sql"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

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

type CharacterUpdateStatus struct {
	ID          int64
	CharacterID int32
	SectionID   CharacterSection
	UpdatedAt   time.Time
	ContentHash string
}
