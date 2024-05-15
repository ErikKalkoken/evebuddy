package model

import "time"

type UpdateSection string

// Updated character sections
const (
	UpdateSectionMail          UpdateSection = "mail"
	UpdateSectionMyCharacter   UpdateSection = "my_character"
	UpdateSectionSkillqueue    UpdateSection = "skillqueue"
	UpdateSectionWalletJournal UpdateSection = "wallet_journal"
)

type MyCharacterUpdateStatus struct {
	MyCharacterID int32
	SectionID     UpdateSection
	UpdatedAt     time.Time
	ContentHash   string
}
