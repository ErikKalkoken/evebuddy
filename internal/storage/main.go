package storage

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Open the database (needs to be called once)
func Open() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("storage.sqlite"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	log.Println("Connected to database")

	// Migrate the schema
	err = db.AutoMigrate(&Character{}, &Token{}, &EveEntity{}, &MailHeader{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
