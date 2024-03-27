package model_test

import (
	"example/esiapp/internal/model"
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Initialize the test database for this test package
	db, err := model.InitDB(":memory:", false)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	os.Exit(m.Run())
}
