package model

import "time"

type UpdateSection string

// Updated character sections
const (
	UpdateSectionMail          UpdateSection = "mail"
	UpdateSectionMailLists     UpdateSection = "mail_list"
	UpdateSectionMailLabels    UpdateSection = "mail_label"
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
