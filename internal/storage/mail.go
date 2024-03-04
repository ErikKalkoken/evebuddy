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

// Return mail IDs of existing mail for a character
func FetchMailIDs(characterId int32) ([]int32, error) {
	var headers []MailHeader
	err := db.Select("mail_id").Where("character_id = ?", characterId).Find(&headers).Error
	if err != nil {
		return nil, err
	}
	var ids []int32
	for _, header := range headers {
		ids = append(ids, header.MailID)
	}
	return ids, nil
}
