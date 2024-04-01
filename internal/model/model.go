// Package models contains all models for persistent storage.
// No direct DB access allowed outside this package.
// This package should not access any other internal packages, except helpers.
package model

import (
	"fmt"
	"log/slog"
	"net/url"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var db *sqlx.DB

var schema = `
	CREATE TABLE IF NOT EXISTS eve_entities (
		id integer PRIMARY KEY NOT NULL,
		category text NOT NULL,
		name text NOT NULL
	);
	CREATE INDEX IF NOT EXISTS eve_entities_name_idx ON eve_entities (name);
	CREATE INDEX IF NOT EXISTS eve_entities_category_idx ON eve_entities (category);

	CREATE TABLE IF NOT EXISTS characters (
		id integer PRIMARY KEY NOT NULL,
		name text NOT NULL,
		corporation_id integer NOT NULL,
		FOREIGN KEY (corporation_id) REFERENCES eve_entities(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS mails_timestamp_idx ON characters (name ASC);

	CREATE TABLE IF NOT EXISTS mail_lists (
		character_id integer NOT NULL,
		eve_entity_id integer NOT NULL,
		FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
		FOREIGN KEY (eve_entity_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
		UNIQUE (character_id, eve_entity_id)
	);

	CREATE TABLE IF NOT EXISTS mail_labels (
		id integer PRIMARY KEY AUTOINCREMENT,
		character_id integer NOT NULL,
		color text NOT NULL,
		label_id integer NOT NULL,
		name text NOT NULL,
		unread_count integer NOT NULL,
		FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE
		UNIQUE (character_id, label_id)
	);

	CREATE TABLE IF NOT EXISTS mail_recipients (
		mail_id integer NOT NULL,
		eve_entity_id integer NOT NULL,
		PRIMARY KEY (mail_id,eve_entity_id),
		FOREIGN KEY (mail_id) REFERENCES mails(id) ON DELETE CASCADE,
		FOREIGN KEY (eve_entity_id) REFERENCES eve_entities(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS mails (
		id integer PRIMARY KEY AUTOINCREMENT,
		body text NOT NULL,
		character_id integer NOT NULL,
		from_id integer NOT NULL,
		is_read numeric NOT NULL,
		mail_id integer NOT NULL,
		subject text NOT NULL,
		timestamp datetime NOT NULL,
		FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
		FOREIGN KEY (from_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
		UNIQUE (character_id, mail_id)
	);
	CREATE INDEX IF NOT EXISTS mails_timestamp_idx ON mails (timestamp DESC);

	CREATE TABLE IF NOT EXISTS mail_mail_labels (
		mail_label_id integer NOT NULL,
		mail_id integer NOT NULL,
		PRIMARY KEY (mail_label_id,mail_id),
		FOREIGN KEY (mail_label_id) REFERENCES mail_labels(id) ON DELETE CASCADE,
		FOREIGN KEY (mail_id) REFERENCES mails(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS tokens (
		access_token text NOT NULL,
		character_id integer PRIMARY KEY NOT NULL,
		expires_at datetime NOT NULL,
		refresh_token text NOT NULL,
		token_type text NOT NULL,
		FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE
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
