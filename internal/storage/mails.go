package storage

import "time"

// An Eve mail belonging to a character
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

// Save creates or updates a mail
func (m *Mail) Save() error {
	err := db.Where("character_id = ? AND mail_id = ?", m.CharacterID, m.MailID).Save(m).Error
	return err
}

// FetchMailIDs return mail IDs of all existing mails for a character
func FetchMailIDs(characterId int32) ([]int32, error) {
	var objs []Mail
	err := db.Select("mail_id").Where("character_id = ?", characterId).Find(&objs).Error
	if err != nil {
		return nil, err
	}
	var ids []int32
	for _, header := range objs {
		ids = append(ids, header.MailID)
	}
	return ids, nil
}

// FetchAllMails returns all mails for a character
func FetchAllMails(characterId int32) ([]Mail, error) {
	var objs []Mail
	err := db.Preload("From").Where("character_id = ?", characterId).Order("time_stamp desc").Find(&objs).Error
	if err != nil {
		return nil, err
	}
	return objs, nil
}
