package storage

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// An Eve mail belonging to a character
type Mail struct {
	gorm.Model
	Body        string
	CharacterID int32
	Character   Character
	FromID      int32
	From        EveEntity
	Labels      []MailLabel `gorm:"many2many:mail_mail_labels;"`
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

// FetchMailsForLabel returns a character's mails for a label
func FetchMailsForLabel(characterID int32, labelID int32) ([]Mail, error) {
	var mm []Mail

	var err error
	if labelID == 0 {
		err = db.Preload("From").Where("character_id = ?", characterID).Order("time_stamp desc").Find(&mm).Error
	} else {
		var l MailLabel
		db.First(&l, labelID)
		err = db.Preload("From").Where("character_id = ?", characterID).Order("time_stamp desc").Model(&l).Association("Mails").Find(&mm)
	}
	if err != nil {
		return nil, err
	}

	return mm, nil
}

func Test() {

	var l MailLabel
	db.First(&l, 4)

	var mm []Mail
	err := db.Model(&l).Association("Mails").Find(&mm)
	log.Print(err)
	log.Print(mm)
	log.Print(len(mm))

	panic("Stop")
}
