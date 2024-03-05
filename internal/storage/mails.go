package storage

import "time"

// A mail belonging to a character
type Mail struct {
	Body        string
	CharacterID int32
	Character   Character
	FromID      int32
	From        EveEntity
	MailID      int32
	IsRead      bool
	Subject     string
	TimeStamp   time.Time
}

func (m *Mail) Save() error {
	err := db.Where("character_id = ? AND mail_id = ?", m.CharacterID, m.MailID).Save(m).Error
	return err
}

// Return mail IDs of existing mail for a character
func FetchMailIDs(characterId int32) ([]int32, error) {
	var headers []Mail
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

func FetchMail(characterId int32) ([]Mail, error) {
	var headers []Mail
	err := db.Preload("From").Where("character_id = ?", characterId).Order("time_stamp desc").Find(&headers).Error
	if err != nil {
		return nil, err
	}
	return headers, nil
}
