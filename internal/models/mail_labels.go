package models

import "fmt"

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

// Save creates or updates a mail label
func (l *MailLabel) Save() error {
	// err := db.Where("character_id = ? AND label_id = ?", l.CharacterID, l.LabelID).Save(l).Error
	// return err
	if l.Character.ID == 0 {
		return fmt.Errorf("can not save mail label without character")
	}
	l.CharacterID = l.Character.ID
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
	var ll []MailLabel
	// err := db.Where("character_id = ? AND id IN (?)", characterID, labelIDs).Find(&ll).Error
	// if err != nil {
	// 	return nil, err
	// }
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

func UpdateOrCreateMailLabel(characterID int32, labelID int32, color string, name string, unreadCount int32) (*MailLabel, error) {
	var l MailLabel
	// err := db.Where("character_id = ? AND label_id = ?", characterID, labelID).Find(&l).Error
	// if err != nil {
	// 	return nil, err
	// }
	// l.CharacterID = characterID
	// l.LabelID = labelID
	// l.Color = color
	// l.Name = name
	// l.UnreadCount = unreadCount
	// err = l.Save()
	// if err != nil {
	// 	return nil, err
	// }
	// slog.Info("Updated mail label", "ID", l.ID, "name", l.Name)
	return &l, nil
}
