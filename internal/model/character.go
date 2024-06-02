// Package model contains the entity objects, which are used across the app.
package model

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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
	ID   int32
	Name string
}

// Updated character sections
const (
	CharacterSectionAssets             CharacterSection = "assets"
	CharacterSectionAttributes         CharacterSection = "attributes"
	CharacterSectionImplants           CharacterSection = "implants"
	CharacterSectionJumpClones         CharacterSection = "jump_clones"
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
	CharacterSectionAssets,
	CharacterSectionAttributes,
	CharacterSectionImplants,
	CharacterSectionJumpClones,
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

type CharacterSection string

func (cs CharacterSection) Name() string {
	t := strings.ReplaceAll(string(cs), "_", " ")
	c := cases.Title(language.English)
	t = c.String(t)
	return t
}

func (cs CharacterSection) CalcContentHash(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	b2 := md5.Sum(b)
	hash := hex.EncodeToString(b2[:])
	return hash, nil
}

// Timeout returns the time until the data of an update section becomes stale.
func (cs CharacterSection) Timeout() time.Duration {
	m := map[CharacterSection]time.Duration{
		CharacterSectionAssets:             3600 * time.Second,
		CharacterSectionAttributes:         120 * time.Second,
		CharacterSectionImplants:           120 * time.Second,
		CharacterSectionJumpClones:         120 * time.Second,
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
	duration, ok := m[cs]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", cs)
		duration = defaultUpdateSectionTimeout
	}
	return duration
}

type CharacterUpdateStatus struct {
	ID            int64
	CharacterID   int32
	ErrorMessage  string
	Section       CharacterSection
	LastUpdatedAt time.Time
	ContentHash   string
}

func (cus CharacterUpdateStatus) IsOK() bool {
	return cus.ErrorMessage == ""
}
