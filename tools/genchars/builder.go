package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

const (
	assetItemsPerLocation      = 100
	assetTypes                 = 10
	implants                   = 8
	jumpClones                 = 2
	locations                  = 5
	mailEntities               = 50
	mails                      = 1000
	notifications              = 1000
	skillqueue                 = 2
	skills                     = 5
	skillGroups                = 2
	walletJournalEntries       = 1000
	walletJournalEntryEntities = 25
	walletTransactionClients   = 25
	walletTransactions         = 1000
	walletTransactionTypes     = 15
)

type CharacterBuilder struct {
	Factor int

	c           *app.Character
	f           *testutil.Factory
	locationIDs []int64
	st          *storage.Storage
}

func NewCharacterBuilder(f *testutil.Factory, st *storage.Storage) *CharacterBuilder {
	b := &CharacterBuilder{
		Factor:      1,
		f:           f,
		locationIDs: make([]int64, 0),
		st:          st,
	}
	return b
}

func (b *CharacterBuilder) Create() {
	// must be first
	b.createTypes()
	b.createCharacter()
	b.createLocations()
	// any order
	b.createAttributes()
	b.createImplants()
	b.createAssets()
	b.createJumpClones()
	b.createMail()
	b.createNotifications()
	b.createSkills()
	b.createSkillqueue()
	b.createWalletJournal()
	b.createWalletTransactions()
	// should be last
	b.setCharacterSections()
	b.setGeneralSections()
	fmt.Printf("COMPLETED: %s\n", b.c.EveCharacter.Name)
}

func (b *CharacterBuilder) createTypes() {
	ctx := context.Background()
	_, err := b.st.GetEveCategory(ctx, app.EveCategorySkill)
	if err == nil {
		return
	}
	if !errors.Is(err, storage.ErrNotFound) {
		panic(err)
	}
	b.f.CreateEveCategory(storage.CreateEveCategoryParams{
		ID:   app.EveCategorySkill,
		Name: "Skill",
	})
}

func (b *CharacterBuilder) createAssets() {
	randomTypeID := b.makeRandomTypes(assetTypes * b.Factor)
	for i, locationId := range b.locationIDs {
		for range assetItemsPerLocation * b.Factor {
			b.f.CreateCharacterAsset(storage.CreateCharacterAssetParams{
				CharacterID: b.c.ID,
				LocationID:  locationId,
				EveTypeID:   randomTypeID(),
			})
		}
		printProgress("assets", len(b.locationIDs), i)
	}
	printSummary("assets", len(b.locationIDs)*assetItemsPerLocation*b.Factor)
}

func (b *CharacterBuilder) createAttributes() {
	b.f.CreateCharacterAttributes(storage.UpdateOrCreateCharacterAttributesParams{CharacterID: b.c.ID})
	fmt.Println("Created attributes")
}
func (b *CharacterBuilder) createCharacter() {
	a := b.f.CreateEveEntityAlliance()
	ec := b.f.CreateEveCharacter(storage.CreateEveCharacterParams{
		AllianceID: a.ID,
	})
	b.c = b.f.CreateCharacter(storage.UpdateOrCreateCharacterParams{
		ID: ec.ID,
	})
	fmt.Printf("Creating new character %s\n", b.c.EveCharacter.Name)
}

func (b *CharacterBuilder) createImplants() {
	for range implants * b.Factor {
		b.f.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: b.c.ID})
	}
	printSummary("jump implants", implants*b.Factor)
}

func (b *CharacterBuilder) createLocations() {
	for i := range locations * b.Factor {
		l := b.f.CreateLocationStructure()
		b.locationIDs = append(b.locationIDs, l.ID)
		printProgress("locations", locations*b.Factor, i)
	}
	printSummary("locations", locations*b.Factor)
}

func (b *CharacterBuilder) randomLocationID() int64 {
	return b.locationIDs[rand.IntN(len(b.locationIDs))]
}

func (b *CharacterBuilder) createJumpClones() {
	// jump clones
	ii := make([]int32, 0)
	for range implants * b.Factor {
		i := b.f.CreateEveType()
		ii = append(ii, i.ID)
	}
	for range jumpClones * b.Factor {
		b.f.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: b.c.ID,
			Implants:    ii,
		})
	}
	printSummary("jump clones", jumpClones*b.Factor)
}

func (b *CharacterBuilder) createMail() {
	labelIDs := []int32{app.MailLabelInbox, app.MailLabelCorp, app.MailLabelAlliance}
	for _, l := range labelIDs {
		b.f.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: b.c.ID,
			LabelID:     l,
		})
	}
	randomLabelID := func() int32 {
		return labelIDs[rand.IntN(len(labelIDs))]
	}
	randomEntityID := b.makeRandomEntities(mailEntities * b.Factor)
	for i := range mails * b.Factor {
		labelID := randomLabelID()
		var recipientID int32
		switch labelID {
		case app.MailLabelCorp:
			recipientID = b.c.EveCharacter.Corporation.ID
		case app.MailLabelAlliance:
			recipientID = b.c.EveCharacter.Alliance.ID
		default:
			recipientID = randomEntityID()
		}
		b.f.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID:  b.c.ID,
			FromID:       randomEntityID(),
			RecipientIDs: []int32{recipientID},
			LabelIDs:     []int32{labelID},
		})
		printProgress("mails", mails*b.Factor, i)
	}
	printSummary("mails", mails*b.Factor)
}

func (b *CharacterBuilder) createNotifications() {
	sender2 := b.f.CreateEveEntityCorporation()
	for i := range notifications * b.Factor {
		b.f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: b.c.ID,
			SenderID:    sender2.ID,
		})
		printProgress("notifications", notifications*b.Factor, i)
	}
	printSummary("notifications", notifications*b.Factor)
}

func (b *CharacterBuilder) createSkills() {
	groupIDs := make([]int32, 0)
	for range skillGroups * b.Factor {
		eg := b.f.CreateEveGroup(storage.CreateEveGroupParams{
			CategoryID:  app.EveCategorySkill,
			IsPublished: true,
		})
		groupIDs = append(groupIDs, eg.ID)
	}
	randomGroupIDs := func() int32 {
		return groupIDs[rand.IntN(len(groupIDs))]
	}
	for i := range skills * b.Factor {
		et := b.f.CreateEveType(storage.CreateEveTypeParams{
			GroupID:     randomGroupIDs(),
			IsPublished: true,
		})
		b.f.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: b.c.ID,
			EveTypeID:   et.ID,
		})
		printProgress("skills", skills*b.Factor, i)
	}
	printSummary("skills", skills*b.Factor)
}

func (b *CharacterBuilder) createSkillqueue() {
	for i := range skillqueue * b.Factor {
		b.f.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: b.c.ID,
		})
		printProgress("skillqueue", skillqueue*b.Factor, i)
	}
	printSummary("skillqueue entries", skillqueue*b.Factor)

}

func (b *CharacterBuilder) createWalletJournal() {
	randomEntityID := b.makeRandomEntities(walletJournalEntryEntities * b.Factor)
	for i := range walletJournalEntries * b.Factor {
		b.f.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			CharacterID:   b.c.ID,
			FirstPartyID:  randomEntityID(),
			SecondPartyID: randomEntityID(),
			TaxReceiverID: randomEntityID(),
		})
		printProgress("wallet journal", walletJournalEntries*b.Factor, i)
	}
	printSummary("wallet journal entries", walletJournalEntries*b.Factor)
}

func (b *CharacterBuilder) createWalletTransactions() {
	randomTypeID := b.makeRandomTypes(walletTransactionTypes * b.Factor)
	randomEntityID := b.makeRandomEntities(walletTransactionClients * b.Factor)
	for i := range walletTransactions * b.Factor {
		var isBuy bool
		if rand.Float32() > 0.5 {
			isBuy = true
		}
		b.f.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{
			IsBuy:       isBuy,
			ClientID:    randomEntityID(),
			CharacterID: b.c.ID,
			LocationID:  b.randomLocationID(),
			EveTypeID:   randomTypeID(),
		})
		printProgress("wallet transactions", walletTransactions*b.Factor, i)
	}
	printSummary("wallet transactions", walletTransactions*b.Factor)
}

func (b *CharacterBuilder) makeRandomTypes(n int) func() int32 {
	typeIDs := make([]int32, 0)
	for range n {
		et := b.f.CreateEveType()
		typeIDs = append(typeIDs, et.ID)
	}
	return func() int32 {
		return typeIDs[rand.IntN(len(typeIDs))]
	}
}

func (b *CharacterBuilder) makeRandomEntities(n int) func() int32 {
	typeIDs := make([]int32, 0)
	for range n {
		et := b.f.CreateEveEntity()
		typeIDs = append(typeIDs, et.ID)
	}
	return func() int32 {
		return typeIDs[rand.IntN(len(typeIDs))]
	}
}

func (b *CharacterBuilder) setCharacterSections() {
	for _, s := range app.CharacterSections {
		b.f.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: b.c.ID,
			Section:     s,
		})
	}
}

func (b *CharacterBuilder) setGeneralSections() {
	for _, s := range app.GeneralSections {
		b.f.CreateGeneralSectionStatus(testutil.GeneralSectionStatusParams{
			Section: s,
		})
	}
}

func printProgress(s string, t, c int) {
	fmt.Printf("%s: %3d%%\r", s, int(float64(c)/float64(t)*100))
}

func printSummary(s string, n int) {
	fmt.Printf("Created %5d %s\n", n, s)
}
