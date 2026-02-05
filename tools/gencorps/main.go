package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/antihax/goesi"
	"github.com/icrowley/fake"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

const (
	corporationID   = 98267621 // RABIS
	systemAbune     = 30004984
	systemEnaluri   = 30045339
	systemJita      = 30000142
	typeAstrahus    = 35832
	typeKeepstar    = 35834
	typeRaitaru     = 35825
	typeTatara      = 35836
	typeAthanor     = 35835
	typeMetanox     = 81826
	structuresCount = 10
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
	typeIDs := []int32{typeAstrahus, typeKeepstar, typeRaitaru, typeTatara, typeAthanor, typeMetanox}
	systemIDs := []int32{systemAbune, systemEnaluri, systemJita}

	for _, id := range typeIDs {
		if _, err := eus.GetOrCreateTypeESI(ctx, id); err != nil {
			log.Fatal(err)
		}
	}
	systems := make(map[int32]*app.EveSolarSystem)
	for _, id := range systemIDs {
		es, err := eus.GetOrCreateSolarSystemESI(ctx, id)
		if err != nil {
			log.Fatal(err)
		}
		systems[id] = es
	}
	corporation, err := st.GetCorporation(ctx, corporationID)
	if errors.Is(err, app.ErrNotFound) {
		log.Fatal("RABIS not found")
	} else if err != nil {
		log.Fatal(err)
	}

	ids, err := st.ListEveLocationIDs(ctx)
	if err != nil {
		log.Fatal(err)
	}
	maxID := set.Max(ids)

	for i := range int64(structuresCount) {
		id := maxID + i + 1
		systemID := systemIDs[rand.IntN(len(systemIDs))]
		typeID := typeIDs[rand.IntN(len(typeIDs))]
		st.UpdateOrCreateCorporationStructure(ctx, storage.UpdateOrCreateCorporationStructureParams{
			CorporationID: corporationID,
			Name:          fmt.Sprintf("%s - %s", systems[systemID].Name, fake.City()),
			State:         app.StructureStateShieldVulnerable,
			StructureID:   id,
			SystemID:      systemID,
			TypeID:        typeID,
			FuelExpires:   optional.New(time.Now().Add(time.Duration(rand.IntN(100)+3) * time.Hour)),
		})
	}

	fmt.Printf("Added %d structures to %s\n", structuresCount, corporation.EveCorporation.Name)
}
