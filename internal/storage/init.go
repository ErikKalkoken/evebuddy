// Package storage contains all models for persistent storage.
// All DB access is abstracted through receivers and helper functions.
// This package should not access any other internal packages, except helpers.
package storage

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

// Initialize the database (needs to be called once)
func Initialize() error {
	myDb, err := gorm.Open(sqlite.Open("storage.sqlite"), &gorm.Config{})
	if err != nil {
		return err
	}
	log.Println("Connected to database")

	err = myDb.AutoMigrate(&Character{}, &Token{}, &EveEntity{}, &MailHeader{})
	if err != nil {
		return err
	}
	db = myDb
	return nil
}
