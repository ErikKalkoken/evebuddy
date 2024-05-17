package model

import "time"

type UpdateSection string

// Updated character sections
const (
	UpdateSectionHome          UpdateSection = "home"
	UpdateSectionMails         UpdateSection = "mails"
	UpdateSectionMailLists     UpdateSection = "mail_lists"
	UpdateSectionMailLabels    UpdateSection = "mail_labels"
	UpdateSectionLocation      UpdateSection = "location"
	UpdateSectionOnline        UpdateSection = "online"
	UpdateSectionShip          UpdateSection = "ship"
	UpdateSectionSkills        UpdateSection = "skills"
	UpdateSectionSkillqueue    UpdateSection = "skillqueue"
	UpdateSectionWalletJournal UpdateSection = "wallet_journal"
	UpdateSectionWalletBalance UpdateSection = "wallet_balance"
)

type MyCharacterUpdateStatus struct {
	MyCharacterID int32
	SectionID     UpdateSection
	UpdatedAt     time.Time
	ContentHash   string
}
