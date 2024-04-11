package model

import "fmt"

type MailList struct {
	ID          uint64
	CharacterID int32 `db:"character_id"`
	Character   Character
	EveEntityID int32 `db:"eve_entity_id"`
	EveEntity   EveEntity
}

// Save creates or updates a mail label.
func (l *MailList) CreateIfNew() error {
	if l.Character.ID != 0 {
		l.CharacterID = l.Character.ID
	}
	if l.CharacterID == 0 {
		return fmt.Errorf("CharacterID can not be zero")
	}
	if l.EveEntity.ID != 0 {
		l.EveEntityID = l.EveEntity.ID
	}
	if l.EveEntityID == 0 {
		return fmt.Errorf("EveEntityID can not be zero")
	}
	r, err := db.NamedExec(`
		INSERT OR IGNORE INTO mail_lists (
			character_id,
			eve_entity_id
		)
		VALUES (
			:character_id,
			:eve_entity_id
		);`,
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

func FetchMailList(characterID int32, entityID int32) (MailList, error) {
	var l MailList
	err := db.Get(
		&l,
		"SELECT * FROM mail_lists WHERE character_id = ? AND eve_entity_id = ?",
		characterID,
		entityID,
	)
	if err != nil {
		return l, err
	}
	return l, nil
}

func FetchAllMailLists(characterID int32) ([]MailList, error) {
	var ll []MailList
	rows, err := db.Query(
		`SELECT *
		FROM mail_lists
		JOIN eve_entities ON eve_entities.id = mail_lists.eve_entity_id
		WHERE character_id = ?
		ORDER by eve_entities.name;`,
		characterID,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var l MailList
		err := rows.Scan(
			&l.CharacterID,
			&l.EveEntityID,
			&l.EveEntity.ID,
			&l.EveEntity.Category,
			&l.EveEntity.Name,
		)
		if err != nil {
			return nil, err
		}
		ll = append(ll, l)
	}
	return ll, nil
}
