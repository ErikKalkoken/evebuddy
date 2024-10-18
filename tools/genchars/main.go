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

const (
	assetItemsPerLocation = 1000
	assetLocations        = 30
	walletJournalEntries  = 1000
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

	// assets
	for i := range assetLocations {
		l := f.CreateLocationStructure()
		ca := f.CreateCharacterAsset(storage.CreateCharacterAssetParams{
			CharacterID: c.ID,
			LocationID:  l.ID,
		})
		for range assetItemsPerLocation - 1 {
			f.CreateCharacterAsset(storage.CreateCharacterAssetParams{
				CharacterID: c.ID,
				LocationID:  ca.LocationID,
				EveTypeID:   ca.EveType.ID,
			})
		}
		printProgress("assets", assetLocations, i)
	}
	fmt.Printf("Created %d assets\n", assetLocations*assetItemsPerLocation)

	// wallet journal
	first := f.CreateEveEntityCharacter()
	second := f.CreateEveEntityCharacter()
	tax := f.CreateEveEntityCharacter()
	for i := range walletJournalEntries {
		f.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			CharacterID:   c.ID,
			FirstPartyID:  first.ID,
			SecondPartyID: second.ID,
			TaxReceiverID: tax.ID,
		})
		printProgress("wallet journal", walletJournalEntries, i)
	}
	fmt.Printf("Created %d wallet journal entries\n", walletJournalEntries)

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
	fmt.Printf("%s: %d%%\r", s, int(float64(c)/float64(t)*100))
}
