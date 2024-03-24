package storage

import (
	"log/slog"
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
	IsRead      bool
	MailID      int32
	Recipients  []EveEntity `gorm:"many2many:mail_recipients;"`
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

	tx := db.Preload("From").Where("character_id = ?", characterID).Order("time_stamp desc")
	var err error
	if labelID == 0 {
		err = tx.Find(&mm).Error
	} else {
		var l MailLabel
		db.First(&l, labelID)
		err = tx.Model(&l).Association("Mails").Find(&mm)
	}
	if err != nil {
		return nil, err
	}

	return mm, nil
}

func FetchMailByID(ID uint) (*Mail, error) {
	var m Mail
	err := db.Preload("From").Preload("Recipients").Find(&m, ID).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func Test() {

	var l MailLabel
	db.First(&l, 4)

	var mm []Mail
	err := db.Model(&l).Association("Mails").Find(&mm)
	slog.Info("result", "mails", mm, "mailsCount", len(mm), "error", err)

	panic("Stop")
}
