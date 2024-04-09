package model_test

import (
	"example/evebuddy/internal/model"
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Initialize the test database for this test package
	db, err := model.InitDB(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	os.Exit(m.Run())
}
