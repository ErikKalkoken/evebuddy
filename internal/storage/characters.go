package storage

import (
	"example/esiapp/internal/esi"

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

// PortraitURL returns an image URL for a portrait of a character
func (c *Character) PortraitURL(size int) fyne.URI {
	return esi.CharacterPortraitURL(c.ID, size)
}

// FetchFirstCharacter returns a random character.
func FetchFirstCharacter() (*Character, error) {
	var obj Character
	if err := db.Get(&obj, "SELECT * FROM characters LIMIT 1;"); err != nil {
		return nil, err
	}
	return &obj, nil
}

func FetchCharacter(characterID int32) (*Character, error) {
	var obj Character
	if err := db.Get(&obj, "SELECT * FROM characters WHERE id = ?;", characterID); err != nil {
		return nil, err
	}
	return &obj, nil
}

// FetchAllCharacters returns all characters.
func FetchAllCharacters() ([]Character, error) {
	var objs []Character
	if err := db.Select(&objs, "SELECT * FROM characters ORDER BY name;"); err != nil {
		return nil, err
	}
	return objs, nil
}
