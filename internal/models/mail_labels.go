package models

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type MailLabel struct {
	ID          uint64
	CharacterID int32 `db:"character_id"`
	Character   Character
	Color       string
	LabelID     int32  `db:"label_id"`
	Mails       []Mail // `gorm:"many2many:mail_mail_labels;"`
	Name        string
	UnreadCount int32 `db:"unread_count"`
}

// Save creates or updates a mail label.
func (l *MailLabel) Save() error {
	if l.Character.ID != 0 {
		l.CharacterID = l.Character.ID
	}
	if l.CharacterID == 0 {
		return fmt.Errorf("CharacterID can not be zero")
	}
	r, err := db.NamedExec(`
		INSERT INTO mail_labels (
			character_id,
			color,
			label_id,
			name,
			unread_count
		)
		VALUES (
			:character_id,
			:color,
			:label_id,
			:name,
			:unread_count
		)
		ON CONFLICT (character_id, label_id) DO
		UPDATE SET
			color=:color,
			name=:name,
			unread_count=:unread_count;`,
		*l,
	)
	if err != nil {
		return err
	}
	newID, err := r.LastInsertId()
	if err != nil {
		return err
	}
	l.ID = uint64(newID)
	return nil
}

func FetchMailLabel(characterID int32, labelID int32) (*MailLabel, error) {
	var l MailLabel
	err := db.Get(
		&l,
		"SELECT * FROM mail_labels WHERE character_id = ? AND label_id = ?",
		characterID,
		labelID,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func FetchMailLabels(characterID int32, labelIDs []int32) ([]MailLabel, error) {
	query, args, err := sqlx.In(
		"SELECT * FROM mail_labels WHERE character_id = ? AND label_id IN (?)",
		characterID,
		labelIDs,
	)
	if err != nil {
		return nil, err
	}
	query = db.Rebind(query)
	rows, err := db.Queryx(query, args...)
	if err != nil {
		return nil, err
	}
	var ll []MailLabel
	for rows.Next() {
		var l MailLabel
		err := rows.StructScan(&l)
		if err != nil {
			return nil, err
		}
		ll = append(ll, l)
	}
	return ll, nil
}

func FetchAllMailLabels(characterID int32) ([]MailLabel, error) {
	var ll []MailLabel
	err := db.Select(&ll, "SELECT * FROM mail_labels WHERE character_id = ?", characterID)
	if err != nil {
		return nil, err
	}
	return ll, nil
}
