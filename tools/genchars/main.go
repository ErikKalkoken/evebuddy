package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

var factorFlag = flag.Int("f", 1, "factor to apply to default quantities")
var numberFlag = flag.Int("n", 1, "number of characters to generate")
var randomFlag = flag.Bool("random", false, "whether to apply the factor with randomness (requires f > 1)")

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatal("Missing DB path")
	}
	dbPath := flag.Arg(0)

	// init database
	dsn := "file:" + dbPath
	dbRW, dbRO, err := storage.InitDB("file:" + dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database %s: %s", dsn, err)
	}
	defer dbRW.Close()
	defer dbRO.Close()
	st := storage.New(dbRW, dbRO)
	f := testutil.NewFactory(st, dbRO)

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
