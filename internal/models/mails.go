package models

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
	if m.Character.ID != 0 {
		m.CharacterID = m.Character.ID
	}
	if m.CharacterID == 0 {
		return fmt.Errorf("CharacterID can not be zero")
	}
	if m.From.ID != 0 {
		m.FromID = m.From.ID
	}
	if m.FromID == 0 {
		return fmt.Errorf("FormID can not be zero")
	}
	r, err := db.NamedExec(`
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
		ON CONFLICT (character_id, mail_id) DO
		UPDATE SET
			body=:body,
			from_id=:from_id,
			is_read=:is_read,
			subject=:subject,
			timestamp=:timestamp`,
		*m,
	)
	if err != nil {
		return err
	}
	newID, err := r.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = uint64(newID)
	return nil
}

// FetchMailIDs return mail IDs of all existing mails for a character
func FetchMailIDs(characterID int32) ([]int32, error) {
	var ids []int32
	err := db.Select(&ids, "SELECT mail_id FROM mails WHERE character_id = ?", characterID)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// FetchMailsForLabel returns a character's mails for a label in descending order by timestamp.
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

// FetchMail returns a mail.
func FetchMail(id uint64) (*Mail, error) {
	row := db.QueryRow(
		`SELECT *
		FROM mails
		JOIN eve_entities ON eve_entities.id = mails.from_id
		WHERE mails.id = ?;`,
		id,
	)
	var m Mail
	err := row.Scan(
		&m.ID,
		&m.Body,
		&m.CharacterID,
		&m.FromID,
		&m.IsRead,
		&m.MailID,
		&m.Subject,
		&m.Timestamp,
		&m.From.ID,
		&m.From.Category,
		&m.From.Name,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
	// err := db.Preload("From").Preload("Recipients").Find(&m, ID).Error
	// if err != nil {
	// 	return nil, err
	// }
	// return &m, nil
}

func Test() {

	// var l MailLabel
	// db.First(&l, 4)

	// var mm []Mail
	// err := db.Model(&l).Association("Mails").Find(&mm)
	// slog.Info("result", "mails", mm, "mailsCount", len(mm), "error", err)

	panic("Stop")
}
