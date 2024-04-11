package model

import (
	"database/sql"
	"example/evebuddy/internal/api/images"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
)

// An Eve Online character.
type Character struct {
	AllianceID     sql.NullInt32 `db:"alliance_id"`
	Alliance       EveEntity
	Birthday       time.Time
	CorporationID  int32 `db:"corporation_id"`
	Corporation    EveEntity
	Description    string
	FactionID      sql.NullInt32 `db:"faction_id"`
	Faction        EveEntity
	Gender         string
	ID             int32
	MailUpdatedAt  sql.NullTime `db:"mail_updated_at"`
	Name           string
	SecurityStatus float32         `db:"security_status"`
	SkillPoints    sql.NullInt64   `db:"skill_points"`
	WalletBalance  sql.NullFloat64 `db:"wallet_balance"`
}

// Save updates or creates a character.
func (c *Character) Save() error {
	if c.Corporation.ID != 0 {
		c.CorporationID = c.Corporation.ID
	}
	if c.Alliance.ID != 0 {
		c.AllianceID.Int32 = c.Alliance.ID
		c.AllianceID.Valid = true
	}
	if c.Faction.ID != 0 {
		c.FactionID.Int32 = c.Faction.ID
		c.FactionID.Valid = true
	}
	if c.CorporationID == 0 {
		return fmt.Errorf("CorporationID can not be zero")
	}
	_, err := db.NamedExec(`
		INSERT INTO characters (
			alliance_id,
			birthday,
			corporation_id,
			description,
			gender,
			faction_id,
			id,
			mail_updated_at,
			name,
			security_status,
			skill_points,
			wallet_balance
		)
		VALUES (
			:alliance_id,
			:birthday,
			:corporation_id,
			:description,
			:gender,
			:faction_id,
			:id,
			:mail_updated_at,
			:name,
			:security_status,
			:skill_points,
			:wallet_balance
		)
		ON CONFLICT (id) DO
		UPDATE SET
			alliance_id = :alliance_id,
			corporation_id = :corporation_id,
			description = :description,
			faction_id = :faction_id,
			mail_updated_at = :mail_updated_at,
			name = :name,
			security_status = :security_status,
			skill_points = :skill_points,
			wallet_balance = :wallet_balance;`,
		*c,
	)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes this character with all it's data
func (c *Character) Delete() error {
	_, err := db.Exec("DELETE FROM characters WHERE id = ?", c.ID)
	if err != nil {
		return err
	}
	slog.Info("Deleted character", "ID", c.ID)
	return nil
}

// PortraitURL returns an image URL for a portrait of a character
func (c *Character) PortraitURL(size int) fyne.URI {
	u, _ := images.CharacterPortraitURL(c.ID, size)
	return u
}

func (c *Character) FetchAlliance() error {
	if !c.AllianceID.Valid {
		return nil
	}
	e, err := FetchEveEntityByID(c.AllianceID.Int32)
	if err != nil {
		return err
	}
	c.Alliance = e
	return nil
}

func (c *Character) FetchFaction() error {
	if !c.FactionID.Valid {
		return nil
	}
	e, err := FetchEveEntityByID(c.FactionID.Int32)
	if err != nil {
		return err
	}
	c.Faction = e
	return nil
}

// FetchFirstCharacter returns a random character.
func FetchFirstCharacter() (Character, error) {
	var c Character
	if err := db.Get(&c, "SELECT * FROM characters LIMIT 1;"); err != nil {
		return c, err
	}
	return c, nil
}

func FetchCharacter(characterID int32) (Character, error) {
	row := db.QueryRowx(
		`SELECT
			characters.alliance_id,
			characters.birthday,
			characters.corporation_id,
			characters.description,
			characters.gender,
			characters.faction_id,
			characters.id,
			characters.mail_updated_at,
			characters.name,
			characters.security_status,
			characters.skill_points,
			characters.wallet_balance,
			corporations.id,
			corporations.category,
			corporations.name
		FROM characters
		JOIN eve_entities AS corporations ON corporations.id = characters.corporation_id
		WHERE characters.id = ?;`,
		characterID,
	)
	var c Character
	err := row.Scan(
		&c.AllianceID,
		&c.Birthday,
		&c.CorporationID,
		&c.Description,
		&c.Gender,
		&c.FactionID,
		&c.ID,
		&c.MailUpdatedAt,
		&c.Name,
		&c.SecurityStatus,
		&c.SkillPoints,
		&c.WalletBalance,
		&c.Corporation.ID,
		&c.Corporation.Category,
		&c.Corporation.Name,
	)
	if err != nil {
		return c, err
	}
	return c, nil
}

// FetchAllCharacters returns all characters ordered by name.
func FetchAllCharacters() ([]Character, error) {
	var cc []Character
	rows, err := db.Query(
		`SELECT
			characters.alliance_id,
			characters.birthday,
			characters.corporation_id,
			characters.description,
			characters.gender,
			characters.faction_id,
			characters.id,
			characters.mail_updated_at,
			characters.name,
			characters.security_status,
			characters.skill_points,
			characters.wallet_balance,
			corporations.id,
			corporations.category,
			corporations.name
		FROM characters
		JOIN eve_entities AS corporations ON corporations.id = characters.corporation_id
		ORDER BY characters.name;`,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c Character
		err := rows.Scan(
			&c.AllianceID,
			&c.Birthday,
			&c.CorporationID,
			&c.Description,
			&c.Gender,
			&c.FactionID,
			&c.ID,
			&c.MailUpdatedAt,
			&c.Name,
			&c.SecurityStatus,
			&c.SkillPoints,
			&c.WalletBalance,
			&c.Corporation.ID,
			&c.Corporation.Category,
			&c.Corporation.Name,
		)
		if err != nil {
			return nil, err
		}
		cc = append(cc, c)

	}
	return cc, nil
}

// FetchCharacterIDs returns all existing character IDs.
func FetchCharacterIDs() ([]int32, error) {
	var ids []int32
	err := db.Select(&ids, "SELECT id FROM characters;")
	if err != nil {
		return nil, err
	}
	return ids, nil
}
