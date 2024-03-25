// Package storage contains all models for persistent storage.
// All DB access is abstracted through receivers and helper functions.
// This package should not access any other internal packages, except helpers.
package models

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var db *sqlx.DB

var schema = `
	CREATE TABLE IF NOT EXISTS characters (
		id integer PRIMARY KEY AUTOINCREMENT,
		name text
	);

	CREATE TABLE IF NOT EXISTS mail_labels (
		id integer PRIMARY KEY AUTOINCREMENT,
		character_id integer,
		color text,
		label_id integer,
		name text,
		unread_count integer,
		FOREIGN KEY (character_id) REFERENCES characters(id)
	);

	CREATE TABLE IF NOT EXISTS mail_recipients (
		mail_id integer,
		eve_entity_id integer,
		PRIMARY KEY (mail_id,eve_entity_id),
		FOREIGN KEY (mail_id) REFERENCES mails(id),
		FOREIGN KEY (eve_entity_id) REFERENCES eve_entities(id)
	);

	CREATE TABLE IF NOT EXISTS mails (
		id integer PRIMARY KEY AUTOINCREMENT,
		body text,
		character_id integer,
		from_id integer,
		is_read numeric,
		mail_id integer,
		subject text,
		timestamp datetime,
		FOREIGN KEY (character_id) REFERENCES characters(id),
		FOREIGN KEY (from_id) REFERENCES eve_entities(id),
		UNIQUE (character_id, mail_id)
	);

	CREATE TABLE IF NOT EXISTS mail_mail_labels (
		mail_label_id integer,
		mail_id integer,
		PRIMARY KEY (mail_label_id,mail_id),
		FOREIGN KEY (mail_label_id) REFERENCES mail_labels(id),
		FOREIGN KEY (mail_id) REFERENCES mails(id)
	);

	CREATE TABLE IF NOT EXISTS eve_entities (
		id integer PRIMARY KEY AUTOINCREMENT,
		category text,
		name text
	);

	CREATE TABLE IF NOT EXISTS tokens (
		access_token text,
		character_id integer PRIMARY KEY,
		expires_at datetime,
		refresh_token text,
		token_type text,
		FOREIGN KEY (character_id) REFERENCES characters(id)
	);
`

// Initialize initializes the database (needs to be called once).
func Initialize(dataSourceName string) error {
	myDb, err := sqlx.Connect("sqlite3", dataSourceName)
	if err != nil {
		return err
	}
	slog.Info("Connected to database")
	_, err = myDb.Exec(schema)
	if err != nil {
		return err
	}
	db = myDb
	return nil
}

// TruncateTables will purge data from all tables. This is meant for tests.
func TruncateTables() {
	sql := `
		DELETE FROM mails;
		DELETE FROM tokens;
		DELETE FROM characters;
		DELETE FROM eve_entities;
	`
	db.MustExec(sql)
	sql = `
		DELETE FROM SQLITE_SEQUENCE WHERE name='mails';
		DELETE FROM SQLITE_SEQUENCE WHERE name='tokens';
		DELETE FROM SQLITE_SEQUENCE WHERE name='characters';
		DELETE FROM SQLITE_SEQUENCE WHERE name='eve_entities';
	`
	db.MustExec(sql)
}
