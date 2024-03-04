package storage

import "time"

// A mail header belonging to a character
type MailHeader struct {
	CharacterID int32
	Character   Character
	FromID      int32
	From        EveEntity
	MailID      int32
	IsRead      bool
	Subject     string
	TimeStamp   time.Time
}

func (m *MailHeader) Save() error {
	err := db.Where("character_id = ? AND mail_id = ?", m.CharacterID, m.MailID).Save(m).Error
	return err
}
