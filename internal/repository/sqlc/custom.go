package sqlc

import (
	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

// Supported categories of EveEntity
const (
	EveEntityAlliance    = "alliance"
	EveEntityCharacter   = "character"
	EveEntityCorporation = "corporation"
	EveEntityFaction     = "faction"
	EveEntityMailList    = "mail_list"
)

//go:embed schema.sql
var schema string

func Schema() string {
	return schema
}
