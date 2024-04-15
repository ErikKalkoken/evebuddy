package repository

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"net/url"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schema string

// ConnectDB initializes the database and returns it.
func ConnectDB(dataSourceName string, create bool) (*sql.DB, error) {
	v := url.Values{}
	v.Add("_fk", "on")
	v.Add("_journal_mode", "WAL")
	v.Add("_synchronous", "normal")
	dsn := fmt.Sprintf("%s?%s", dataSourceName, v.Encode())
	slog.Debug("Connecting to sqlite", "dsn", dsn)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	slog.Info("Connected to database")
	if create {
		_, err = db.Exec(schema)
		if err != nil {
			return nil, err
		}

	}
	return db, nil
}

type Repository struct {
	q  *Queries
	db DBTX
}

func NewRepository(db DBTX) *Repository {
	r := &Repository{q: New(db), db: db}
	return r
}

// TruncateTables will purge data from all tables. This is meant for tests.
func TruncateTables(db *sql.DB) {
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
	db.Exec(sql)
	sql = `
		DELETE FROM SQLITE_SEQUENCE WHERE name='mail_recipients';
		DELETE FROM SQLITE_SEQUENCE WHERE name='mail_mail_labels';
		DELETE FROM SQLITE_SEQUENCE WHERE name='mail_labels';
		DELETE FROM SQLITE_SEQUENCE WHERE name='mails';
		DELETE FROM SQLITE_SEQUENCE WHERE name='tokens';
		DELETE FROM SQLITE_SEQUENCE WHERE name='characters';
		DELETE FROM SQLITE_SEQUENCE WHERE name='eve_entities';
	`
	db.Exec(sql)
}
