// dbperf is for measuring the DB performance.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

const objectsCount = int32(10_000)

func main() {
	dsn := "file:test.db"
	dbRW, dbRO, err := storage.InitDB(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize database %s: %s", dsn, err)
	}
	if err := testutil.TruncateTables(dbRW); err != nil {
		log.Fatal(err)
	}
	st := storage.New(dbRW, dbRO)
	ctx := context.Background()
	start := time.Now()
	for i := range objectsCount {
		st.CreateEveEntity(ctx, storage.CreateEveEntityParams{
			ID:   i + 1,
			Name: fmt.Sprintf("Dummy #%d", i+1),
		})
	}
	dbRW.Close()
	dbRO.Close()
	elapsed := time.Since(start)
	f := float64(objectsCount) / elapsed.Seconds()
	fmt.Printf("Elapsed: %s | throughput per second: %f\n", elapsed, f)
}
