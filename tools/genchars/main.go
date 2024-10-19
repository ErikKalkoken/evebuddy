package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/appdirs"
)

const (
	dbFileName = "evebuddy.sqlite"
)

var factorFlag = flag.Int("f", 1, "factor to apply to default quantities")
var numberFlag = flag.Int("n", 1, "number of characters to generate")
var randomFlag = flag.Bool("random", false, "whether to apply the factor with randomness (requires f > 1)")

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
	fmt.Println()
	for i := range *numberFlag {
		var factor int
		if *randomFlag {
			factor = min(1, int(rand.Float32()*float32(*factorFlag)))
		} else {
			factor = *factorFlag
		}
		b := NewCharacterBuilder(&f, st)
		b.Factor = factor
		b.Create()
		fmt.Printf("Completed character %d / %d\n\n", i+1, *numberFlag)
	}
}
