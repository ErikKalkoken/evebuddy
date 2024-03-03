package storage

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// Open the database (needs to be called once)
func Open() *sql.DB {
	db, err := sql.Open("sqlite3", "./storage.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to database")
	DB = db
	s := `
		CREATE TABLE IF NOT EXISTS tokens (
			access_token text not null,
			character_id integer not null primary key,
			character_name text not null,
			expires_at integer not null,
			refresh_token text not null,
			token_type text not null
		);
	`
	_, err = db.Exec(s)
	if err != nil {
		log.Fatal(err)
	}
	return db
}
