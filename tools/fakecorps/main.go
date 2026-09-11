package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/fnt-eve/goesi-openapi"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/icrowley/fake"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

const (
	corporationRABIS = 98267621 // RABIS
	corporationVREGS = 98394960
	systemAbune      = 30004984
	systemEnaluri    = 30045339
	systemJita       = 30000142
	typeAstrahus     = 35832
	typeKeepstar     = 35834
	typeRaitaru      = 35825
	typeTatara       = 35836
	typeAthanor      = 35835
	typeMetanox      = 81826
	structuresCount  = 10
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

	rhc1 := retryablehttp.NewClient()
	sc := new(statuscache.StatusCache)

	ctx := context.Background()
	if err := sc.Init(ctx, st); err != nil {
		log.Fatal(err)
	}
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		Storage: st,
		ESIClient: goesi.NewESIClientWithOptions(rhc1.StandardClient(), goesi.ClientOptions{
			UserAgent: "EveBuddy/1.0 (test@kalkoken.net)",
		}),
		Signals:            app.NewSignals(),
		StatusCacheService: sc,
	})

	corporationIDs := []int64{corporationRABIS, corporationVREGS}
	typeIDs := []int64{typeAstrahus, typeKeepstar, typeRaitaru, typeTatara, typeAthanor, typeMetanox}
	systemIDs := []int64{systemAbune, systemEnaluri, systemJita}

	for _, id := range typeIDs {
		if _, err := eus.GetOrCreateTypeESI(ctx, id); err != nil {
			log.Fatal(err)
		}
	}
	systems := make(map[int64]*app.EveSolarSystem)
	for _, id := range systemIDs {
		es, err := eus.GetOrCreateSolarSystemESI(ctx, id)
		if err != nil {
			log.Fatal(err)
		}
		systems[id] = es
	}

	ids, err := st.ListEveLocationIDs(ctx)
	if err != nil {
		log.Fatal(err)
	}
	maxID := set.Max(ids)

	for _, corporationID := range corporationIDs {
		corporation, err := st.GetCorporation(ctx, corporationID)
		if errors.Is(err, app.ErrNotFound) {
			log.Printf("corporation %d not found\n", corporationID)
			continue
		}
		if err != nil {
			log.Fatal(err)
		}

		for i := range int64(structuresCount) {
			id := maxID + i + 1
			systemID := systemIDs[rand.IntN(len(systemIDs))]
			typeID := typeIDs[rand.IntN(len(typeIDs))]

			var services []storage.StructureServiceParams
			for i := range rand.IntN(4) {
				services = append(services, storage.StructureServiceParams{
					Name:  fmt.Sprintf("Service%d", i+1),
					State: app.StructureServiceStateOnline,
				})
			}

			err := st.UpdateOrCreateCorporationStructure(ctx, storage.UpdateOrCreateCorporationStructureParams{
				CorporationID: corporationID,
				FuelExpires:   optional.New(time.Now().Add(time.Duration(rand.IntN(100)+3) * time.Hour)),
				Name:          optional.New(fmt.Sprintf("%s - %s", systems[systemID].Name, fake.City())),
				Services:      services,
				State:         app.StructureStateShieldVulnerable,
				StructureID:   id,
				SystemID:      systemID,
				TypeID:        typeID,
			})
			if err != nil {
				log.Fatal(err)
			}
		}

		fmt.Printf("Added %d structures to %s\n", structuresCount, corporation.EveCorporation.Name)
	}
}
