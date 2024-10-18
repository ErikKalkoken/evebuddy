package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/appdirs"
)

const (
	dbFileName = "evebuddy.sqlite"
)

var factorFlag = flag.Int("f", 1, "factor to apply to default quantities")

func main() {
	flag.Parse()

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
	st := storage.New(db)
	f := testutil.NewFactory(st, db)

	// build character
	b := NewCharacterBuilder(&f, st)
	b.Factor = *factorFlag
	b.Create()
}
