package model

import "time"

type UpdateSection string

// Updated character sections
const (
	UpdateSectionMail          = "mail"
	UpdateSectionMyCharacter   = "my_character"
	UpdateSectionSkillqueue    = "skillqueue"
	UpdateSectionWalletJournal = "wallet_journal"
)

type MyCharacterUpdateStatus struct {
	MyCharacterID int32
	SectionID     UpdateSection
	UpdatedAt     time.Time
	ContentHash   string
}
