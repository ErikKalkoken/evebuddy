package model

import "time"

type UpdateSection string

// Updated character sections
const (
	UpdateSectionHome               UpdateSection = "home"
	UpdateSectionMails              UpdateSection = "mails"
	UpdateSectionMailLists          UpdateSection = "mail_lists"
	UpdateSectionMailLabels         UpdateSection = "mail_labels"
	UpdateSectionLocation           UpdateSection = "location"
	UpdateSectionOnline             UpdateSection = "online"
	UpdateSectionShip               UpdateSection = "ship"
	UpdateSectionSkills             UpdateSection = "skills"
	UpdateSectionSkillqueue         UpdateSection = "skillqueue"
	UpdateSectionWalletBalance      UpdateSection = "wallet_balance"
	UpdateSectionWalletJournal      UpdateSection = "wallet_journal"
	UpdateSectionWalletTransactions UpdateSection = "wallet_transactions"
)

type MyCharacterUpdateStatus struct {
	MyCharacterID int32
	SectionID     UpdateSection
	UpdatedAt     time.Time
	ContentHash   string
}
