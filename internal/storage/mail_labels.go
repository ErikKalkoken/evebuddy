package storage

import (
	"gorm.io/gorm"
)

type MailLabel struct {
	gorm.Model
	CharacterID int32
	Character   Character
	Color       string
	ID          int32
	Mails       []Mail `gorm:"many2many:mail_mail_labels;"`
	Name        string
	UnreadCount int32
}

// Save creates or updates a mail label
func (l *MailLabel) Save() error {
	err := db.Where("character_id = ? AND id = ?", l.CharacterID, l.ID).Save(l).Error
	return err
}

func FetchMailLabels(characterID int32, labelIDs []int32) ([]MailLabel, error) {
	var ll []MailLabel
	err := db.Where("character_id = ? AND id IN (?)", characterID, labelIDs).Find(&ll).Error
	if err != nil {
		return nil, err
	}
	return ll, nil
}

func FetchAllMailLabels(characterID int32) ([]MailLabel, error) {
	var ll []MailLabel
	err := db.Where("character_id = ?", characterID).Order("id").Find(&ll).Error
	if err != nil {
		return nil, err
	}
	return ll, nil
}
