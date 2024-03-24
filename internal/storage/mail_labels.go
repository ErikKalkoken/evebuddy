package storage

import (
	"log/slog"

	"gorm.io/gorm"
)

type MailLabel struct {
	gorm.Model
	CharacterID int32
	Character   Character
	Color       string
	LabelID     int32
	Mails       []Mail `gorm:"many2many:mail_mail_labels;"`
	Name        string
	UnreadCount int32
}

// Save creates or updates a mail label
func (l *MailLabel) Save() error {
	err := db.Where("character_id = ? AND label_id = ?", l.CharacterID, l.LabelID).Save(l).Error
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

func UpdateOrCreateMailLabel(characterID int32, labelID int32, color string, name string, unreadCount int32) (*MailLabel, error) {
	var l MailLabel
	err := db.Where("character_id = ? AND label_id = ?", characterID, labelID).Find(&l).Error
	if err != nil {
		return nil, err
	}
	l.CharacterID = characterID
	l.LabelID = labelID
	l.Color = color
	l.Name = name
	l.UnreadCount = unreadCount
	err = l.Save()
	if err != nil {
		return nil, err
	}
	slog.Info("Updated mail label", "ID", l.ID, "name", l.Name)
	return &l, nil
}
