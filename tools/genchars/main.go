package main

import (
	"fmt"
	"log"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
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
	st := storage.New(db)
	f := testutil.NewFactory(st, db)
	c := f.CreateCharacter()
	// wallet journal
	first := f.CreateEveEntityCharacter()
	second := f.CreateEveEntityCharacter()
	tax := f.CreateEveEntityCharacter()
	for i := range 1_000 {
		f.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			CharacterID:   c.ID,
			FirstPartyID:  first.ID,
			SecondPartyID: second.ID,
			TaxReceiverID: tax.ID,
		})
		printProgress("wallet journal", 1_000, i)
	}
	fmt.Println()
	fmt.Println("Created wallet journal entries")
	// assets
	for i := range 30 {
		ca := f.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: c.ID})
		for range 1_000 {
			f.CreateCharacterAsset(storage.CreateCharacterAssetParams{
				CharacterID: c.ID,
				LocationID:  ca.LocationID,
				EveTypeID:   ca.EveType.ID,
			})
		}
		printProgress("assets", 30, i)
	}
	fmt.Println()
	fmt.Println("Created assets")
	// set all sections as loaded
	for _, s := range app.CharacterSections {
		f.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     s,
		})
	}
	fmt.Println("DONE")
}

func printProgress(s string, t, c int) {
	fmt.Printf("\r%s: %d%%", s, int(float64(c)/float64(t)*100))
}
