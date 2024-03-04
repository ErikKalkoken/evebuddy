package storage

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Open the database (needs to be called once)
func Open() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("storage.sqlite"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to database")

	// Migrate the schema
	err = db.AutoMigrate(&Character{}, &Token{})
	if err != nil {
		log.Fatal(err)
	}
	return db
}
