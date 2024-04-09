// Package models contains all models for persistent storage.
// No direct DB access allowed outside this package.
// This package should not access any other internal packages, except helpers.
package model

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var ErrDoesNotExist = errors.New("object does not exist in database")

var db *sqlx.DB

var schema = `
	CREATE TABLE IF NOT EXISTS eve_entities (
		id INTEGER PRIMARY KEY NOT NULL,
		category TEXT NOT NULL,
		name TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS eve_entities_name_idx ON eve_entities (name);
	CREATE INDEX IF NOT EXISTS eve_entities_category_idx ON eve_entities (category);

	CREATE TABLE IF NOT EXISTS characters (
		id INTEGER PRIMARY KEY NOT NULL,
		name TEXT NOT NULL,
		corporation_id INTEGER NOT NULL,
		mail_updated_at DATETIME,
		FOREIGN KEY (corporation_id) REFERENCES eve_entities(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS mails_timestamp_idx ON characters (name ASC);

	CREATE TABLE IF NOT EXISTS mail_lists (
		character_id INTEGER NOT NULL,
		eve_entity_id INTEGER NOT NULL,
		FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
		FOREIGN KEY (eve_entity_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
		UNIQUE (character_id, eve_entity_id)
	);

	CREATE TABLE IF NOT EXISTS mail_labels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		character_id INTEGER NOT NULL,
		color TEXT NOT NULL,
		label_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		unread_count INTEGER NOT NULL,
		FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE
		UNIQUE (character_id, label_id)
	);

	CREATE TABLE IF NOT EXISTS mail_recipients (
		mail_id INTEGER NOT NULL,
		eve_entity_id INTEGER NOT NULL,
		PRIMARY KEY (mail_id,eve_entity_id),
		FOREIGN KEY (mail_id) REFERENCES mails(id) ON DELETE CASCADE,
		FOREIGN KEY (eve_entity_id) REFERENCES eve_entities(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS mails (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		body TEXT NOT NULL,
		character_id INTEGER NOT NULL,
		from_id INTEGER NOT NULL,
		is_read numeric NOT NULL,
		mail_id INTEGER NOT NULL,
		subject TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
		FOREIGN KEY (from_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
		UNIQUE (character_id, mail_id)
	);
	CREATE INDEX IF NOT EXISTS mails_timestamp_idx ON mails (timestamp DESC);

	CREATE TABLE IF NOT EXISTS mail_mail_labels (
		mail_label_id INTEGER NOT NULL,
		mail_id INTEGER NOT NULL,
		PRIMARY KEY (mail_label_id,mail_id),
		FOREIGN KEY (mail_label_id) REFERENCES mail_labels(id) ON DELETE CASCADE,
		FOREIGN KEY (mail_id) REFERENCES mails(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS tokens (
		access_token TEXT NOT NULL,
		character_id INTEGER PRIMARY KEY NOT NULL,
		expires_at DATETIME NOT NULL,
		refresh_token TEXT NOT NULL,
		token_type TEXT NOT NULL,
		FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY NOT NULL,
		value BLOB NOT NULL
	);
`

// InitDB initializes the database (needs to be called once).
func InitDB(dataSourceName string) (*sqlx.DB, error) {
	v := url.Values{}
	v.Add("_fk", "on")
	v.Add("_journal_mode", "WAL")
	v.Add("_synchronous", "normal")
	dsn := fmt.Sprintf("%s?%s", dataSourceName, v.Encode())
	myDb, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	slog.Info("Connected to database")
	_, err = myDb.Exec(schema)
	if err != nil {
		return nil, err
	}
	db = myDb
	return myDb, nil
}

// TruncateTables will purge data from all tables. This is meant for tests.
func TruncateTables() {
	sql := `
		DELETE FROM mail_recipients;
		DELETE FROM mail_mail_labels;
		DELETE FROM mail_labels;
		DELETE FROM mails;
		DELETE FROM tokens;
		DELETE FROM characters;
		DELETE FROM eve_entities;
		DELETE FROM settings;
	`
	db.MustExec(sql)
	sql = `
		DELETE FROM SQLITE_SEQUENCE WHERE name='mail_recipients';
		DELETE FROM SQLITE_SEQUENCE WHERE name='mail_mail_labels';
		DELETE FROM SQLITE_SEQUENCE WHERE name='mail_labels';
		DELETE FROM SQLITE_SEQUENCE WHERE name='mails';
		DELETE FROM SQLITE_SEQUENCE WHERE name='tokens';
		DELETE FROM SQLITE_SEQUENCE WHERE name='characters';
		DELETE FROM SQLITE_SEQUENCE WHERE name='eve_entities';
	`
	db.MustExec(sql)
}
