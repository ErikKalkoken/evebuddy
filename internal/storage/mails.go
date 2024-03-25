package storage

import (
	"fmt"
	"time"
)

// An Eve mail belonging to a character
type Mail struct {
	Body        string
	CharacterID int32 `db:"character_id"`
	Character   Character
	FromID      int32 `db:"from_id"`
	From        EveEntity
	Labels      []MailLabel // `gorm:"many2many:mail_mail_labels;"`
	IsRead      bool        `db:"is_read"`
	ID          uint64
	MailID      int32       `db:"mail_id"`
	Recipients  []EveEntity // `gorm:"many2many:mail_recipients;"`
	Subject     string
	Timestamp   time.Time
}

// Save creates or updates a mail
func (m *Mail) Save() error {
	if m.Character.ID == 0 {
		return fmt.Errorf("can not save mail without character")
	}
	m.CharacterID = m.Character.ID
	if m.From.ID == 0 {
		return fmt.Errorf("can not save mail without from")
	}
	m.FromID = m.From.ID
	_, err := db.NamedExec(`
		INSERT INTO mails (
			body,
			character_id,
			from_id,
			is_read,
			mail_id,
			subject,
			timestamp
		)
		VALUES (
			:body,
			:character_id,
			:from_id,
			:is_read,
			:mail_id,
			:subject,
			:timestamp
		)
		ON CONFLICT (id) DO
		UPDATE SET
			body=:body,
			character_id=:character_id,
			from_id=:from_id,
			is_read=:is_read,
			mail_id=:mail_id,
			subject=:subject,
			timestamp=:timestamp`,
		*m,
	)
	if err != nil {
		return err
	}
	return nil
}

// FetchMailIDs return mail IDs of all existing mails for a character
func FetchMailIDs(characterId int32) ([]int32, error) {
	// var objs []Mail
	// err := db.Select("mail_id").Where("character_id = ?", characterId).Find(&objs).Error
	// if err != nil {
	// 	return nil, err
	// }
	// var ids []int32
	// for _, header := range objs {
	// 	ids = append(ids, header.MailID)
	// }
	// return ids, nil
	return nil, nil
}

// FetchMailsForLabel returns a character's mails for a label
func FetchMailsForLabel(characterID int32, labelID int32) ([]Mail, error) {
	var mm []Mail

	// tx := db.Preload("From").Where("character_id = ?", characterID).Order("time_stamp desc")
	// var err error
	// if labelID == 0 {
	// 	err = tx.Find(&mm).Error
	// } else {
	// 	var l MailLabel
	// 	db.First(&l, labelID)
	// 	err = tx.Model(&l).Association("Mails").Find(&mm)
	// }
	// if err != nil {
	// 	return nil, err
	// }

	return mm, nil
}

func FetchMailByID(ID uint64) (*Mail, error) {
	var m Mail
	// err := db.Preload("From").Preload("Recipients").Find(&m, ID).Error
	// if err != nil {
	// 	return nil, err
	// }
	return &m, nil
}

func Test() {

	// var l MailLabel
	// db.First(&l, 4)

	// var mm []Mail
	// err := db.Model(&l).Association("Mails").Find(&mm)
	// slog.Info("result", "mails", mm, "mailsCount", len(mm), "error", err)

	panic("Stop")
}
