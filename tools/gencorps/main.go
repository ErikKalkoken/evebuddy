package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/antihax/goesi"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

const (
	corporationID = 98267621 // RABIS
	solarSystemID = 30004984 // Abune
	typeAstrahus  = 35832
)

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

	eus := eveuniverseservice.New(eveuniverseservice.Params{
		Storage:   st,
		ESIClient: goesi.NewAPIClient(nil, "EVE Buddy generate"),
	})

	ctx := context.Background()
	if _, err := eus.GetOrCreateCategoryESI(ctx, app.EveCategoryStructure); err != nil {
		log.Fatal(err)
	}
	if _, err := eus.GetOrCreateSolarSystemESI(ctx, solarSystemID); err != nil {
		log.Fatal(err)
	}
	if _, err := st.GetCorporation(ctx, corporationID); errors.Is(err, app.ErrNotFound) {
		log.Fatal("RABIS not found")
	} else if err != nil {
		log.Fatal(err)
	}

	ids, err := st.ListEveLocationIDs(ctx)
	if err != nil {
		log.Fatal(err)
	}
	maxID := set.Max(ids)

	for i := range int64(1) {
		id := maxID + i + 1
		st.UpdateOrCreateCorporationStructure(ctx, storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: corporationID,
			Name:          fmt.Sprintf("Generated #%d", id),
			State:         app.StructureStateShieldVulnerable,
			StructureID:   id,
			SystemID:      solarSystemID,
			TypeID:        typeAstrahus,
			FuelExpires:   optional.New(time.Now().Add(6 * time.Hour)),
		})
	}
}
