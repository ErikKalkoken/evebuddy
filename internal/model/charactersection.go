package model

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const defaultUpdateSectionTimeout = 3600 * time.Second

// Updated character sections
const (
	SectionAssets             CharacterSection = "assets"
	SectionAttributes         CharacterSection = "attributes"
	SectionImplants           CharacterSection = "implants"
	SectionJumpClones         CharacterSection = "jump_clones"
	SectionLocation           CharacterSection = "location"
	SectionMailLists          CharacterSection = "mail_lists"
	SectionMailLabels         CharacterSection = "mail_labels"
	SectionMails              CharacterSection = "mails"
	SectionOnline             CharacterSection = "online"
	SectionShip               CharacterSection = "ship"
	SectionSkills             CharacterSection = "skills"
	SectionSkillqueue         CharacterSection = "skillqueue"
	SectionWalletBalance      CharacterSection = "wallet_balance"
	SectionWalletJournal      CharacterSection = "wallet_journal"
	SectionWalletTransactions CharacterSection = "wallet_transactions"
)

var CharacterSections = []CharacterSection{
	SectionAssets,
	SectionAttributes,
	SectionImplants,
	SectionJumpClones,
	SectionLocation,
	SectionMailLabels,
	SectionMailLists,
	SectionMails,
	SectionOnline,
	SectionShip,
	SectionSkills,
	SectionSkillqueue,
	SectionWalletBalance,
	SectionWalletJournal,
	SectionWalletTransactions,
}

type CharacterSection string

func (cs CharacterSection) DisplayName() string {
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
		SectionAssets:             3600 * time.Second,
		SectionAttributes:         120 * time.Second,
		SectionImplants:           120 * time.Second,
		SectionJumpClones:         120 * time.Second,
		SectionLocation:           30 * time.Second, // 5 seconds min
		SectionMailLabels:         30 * time.Second,
		SectionMailLists:          120 * time.Second,
		SectionMails:              30 * time.Second,
		SectionOnline:             60 * time.Second,
		SectionShip:               30 * time.Second, // 5 seconds min
		SectionSkillqueue:         120 * time.Second,
		SectionSkills:             120 * time.Second,
		SectionWalletBalance:      120 * time.Second,
		SectionWalletJournal:      3600 * time.Second,
		SectionWalletTransactions: 3600 * time.Second,
	}
	duration, ok := m[cs]
	if !ok {
		slog.Warn("Requested duration for unknown section. Using default.", "section", cs)
		duration = defaultUpdateSectionTimeout
	}
	return duration
}
