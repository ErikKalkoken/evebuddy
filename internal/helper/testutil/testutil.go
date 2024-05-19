package testutil

import (
	"database/sql"

	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func New() (*sql.DB, *storage.Storage, Factory) {
	db, err := storage.InitDB(":memory:")
	if err != nil {
		panic(err)
	}
	r := storage.New(db)
	factory := NewFactory(r, db)
	return db, r, factory
}

// TruncateTables will purge data from all tables. This is meant for tests.
func TruncateTables(db *sql.DB) {
	sql := `
		DELETE FROM locations;
		DELETE FROM mail_recipients;
		DELETE FROM mail_mail_labels;
		DELETE FROM mail_labels;
		DELETE FROM mails;
		DELETE FROM character_skills;
		DELETE FROM skillqueue_items;
		DELETE FROM wallet_transactions;
		DELETE FROM wallet_journal_entries;
		DELETE FROM tokens_scopes;
		DELETE FROM scopes;
		DELETE FROM tokens;
		DELETE FROM my_character_update_status;
		DELETE FROM my_characters;
		DELETE FROM eve_characters;
		DELETE FROM eve_entities;
		DELETE FROM eve_categories;
		DELETE FROM eve_groups;
		DELETE FROM eve_types;
		DELETE FROM dictionary;
	`
	db.Exec(sql)
	sql = `
		DELETE FROM SQLITE_SEQUENCE WHERE name='mail_mail_labels';
		DELETE FROM SQLITE_SEQUENCE WHERE name='mail_labels';
		DELETE FROM SQLITE_SEQUENCE WHERE name='mails';
	`
	db.Exec(sql)
}
