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
		panic("failed to connect database")
	}
	log.Println("Connected to database")

	// Migrate the schema
	db.AutoMigrate(&Token{})

	return db
}
