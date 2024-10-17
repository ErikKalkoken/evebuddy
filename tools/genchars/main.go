package main

import (
	"fmt"
	"log"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/appdirs"
)

const (
	dbFileName = "evebuddy.sqlite"
)

func main() {
	// init dirs
	ad, err := appdirs.New()
	if err != nil {
		log.Fatal(err)
	}

	// init database
	dsn := fmt.Sprintf("file:%s/%s", ad.Data, dbFileName)
	db, err := storage.InitDB(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize database %s: %s", dsn, err)
	}
	defer db.Close()
	// st := storage.New(db)
	// ctx := context.Background()

}
