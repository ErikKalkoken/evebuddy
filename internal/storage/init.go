// Package storage contains all models for persistent storage.
// All DB access is abstracted through receivers and helper functions.
// This package should not access any other internal packages, except helpers.
package storage

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

// Initialize initializes the database (needs to be called once).
func Initialize() error {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second * 5, // Slow SQL threshold
			LogLevel:                  logger.Silent,   // Log level
			IgnoreRecordNotFoundError: true,            // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,            // Don't include params in the SQL log
			Colorful:                  false,           // Disable color
		},
	)
	myDb, err := gorm.Open(sqlite.Open("storage.sqlite"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return err
	}
	log.Println("Connected to database")

	err = myDb.AutoMigrate(&Character{}, &Token{}, &EveEntity{}, &Mail{}, &MailLabel{})
	if err != nil {
		return err
	}
	db = myDb
	return nil
}
