package model

import (
	"example/esiapp/internal/api/images"
	"log/slog"

	"fyne.io/fyne/v2"
)

// An Eve Online character.
type Character struct {
	ID   int32
	Name string
}

// Save updates or creates a character.
func (c *Character) Save() error {
	_, err := db.NamedExec(`
		INSERT INTO characters (id, name)
		VALUES (:id, :name)
		ON CONFLICT (id) DO
		UPDATE SET name=:name;`,
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

// FetchFirstCharacter returns a random character.
func FetchFirstCharacter() (*Character, error) {
	var c Character
	if err := db.Get(&c, "SELECT * FROM characters LIMIT 1;"); err != nil {
		return nil, err
	}
	return &c, nil
}

func FetchCharacter(characterID int32) (*Character, error) {
	var c Character
	if err := db.Get(&c, "SELECT * FROM characters WHERE id = ?;", characterID); err != nil {
		return nil, err
	}
	return &c, nil
}

// FetchAllCharacters returns all characters ordered by name.
func FetchAllCharacters() ([]Character, error) {
	var cc []Character
	if err := db.Select(&cc, "SELECT * FROM characters ORDER BY name;"); err != nil {
		return nil, err
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