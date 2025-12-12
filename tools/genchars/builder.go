package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

const (
	assetItemsPerLocation      = 100
	assetTypes                 = 10
	contracts                  = 1
	contractItems              = 1
	implants                   = 2
	jumpClones                 = 2
	locations                  = 5
	mailEntities               = 50
	mailLabels                 = 10
	mailLists                  = 10
	mailMaxRecipients          = 3
	mails                      = 1000
	notifications              = 1000
	skillGroups                = 2
	skillqueue                 = 2
	skills                     = 5
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
	b.createContracts()
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
}

func (b *CharacterBuilder) createTypes() {
	ctx := context.Background()
	_, err := b.st.GetEveCategory(ctx, app.EveCategorySkill)
	if err == nil {
		return
	}
	if !errors.Is(err, app.ErrNotFound) {
		panic(err)
	}
	b.f.CreateEveCategory(storage.CreateEveCategoryParams{
		ID:   app.EveCategorySkill,
		Name: "Skill",
	})
}

func (b *CharacterBuilder) createAssets() {
	randomTypeID := b.makeRandomTypes(assetTypes * b.Factor)
	for i, locationID := range b.locationIDs {
		for range assetItemsPerLocation * b.Factor {
			b.f.CreateCharacterAsset(storage.CreateCharacterAssetParams{
				CharacterID: b.c.ID,
				LocationID:  locationID,
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
	c := b.f.CreateEveEntityCharacter()
	a := b.f.CreateEveEntityAlliance()
	ec := b.f.CreateEveCharacter(storage.CreateEveCharacterParams{
		AllianceID: a.ID,
		Name:       c.Name,
		ID:         c.ID,
	})
	b.c = b.f.CreateCharacterFull(storage.CreateCharacterParams{
		ID: ec.ID,
	})
	b.f.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
	fmt.Printf("Creating new character %s with factor %d\n", b.c.EveCharacter.Name, b.Factor)
}

func (b *CharacterBuilder) createContracts() {
	makeTypeID := b.makeRandomTypes(assetTypes * b.Factor)
	var count int
	for range contracts * b.Factor {
		o := b.f.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: b.c.ID,
			Type:        app.ContractTypeItemExchange,
			Price:       float64(rand.IntN(10_000_000*100)) / 100,
		})
		for range contractItems {
			b.f.CreateCharacterContractItem(storage.CreateCharacterContractItemParams{
				ContractID: o.ID,
				TypeID:     makeTypeID(),
				IsIncluded: true,
			})
		}
		count++
	}
	printSummary("contracts", count)
}

func (b *CharacterBuilder) createImplants() {
	for range implants * b.Factor {
		b.f.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: b.c.ID})
	}
	printSummary("jump implants", implants*b.Factor)
}

func (b *CharacterBuilder) createLocations() {
	for i := range locations * b.Factor {
		l := b.f.CreateEveLocationStructure()
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
	labelIDs := []int32{app.MailLabelInbox, app.MailLabelSent, app.MailLabelCorp, app.MailLabelAlliance}
	for _, l := range labelIDs {
		b.f.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: b.c.ID,
			LabelID:     l,
		})
	}
	for range mailLabels * b.Factor {
		ml := b.f.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: b.c.ID,
		})
		labelIDs = append(labelIDs, ml.LabelID)
	}
	randomLabelID := func() int32 {
		return labelIDs[rand.IntN(len(labelIDs))]
	}
	listIDs := make([]int32, 0)
	for range mailLists * b.Factor {
		ml := b.f.CreateCharacterMailList(b.c.ID)
		listIDs = append(listIDs, ml.ID)
	}
	randomListID := func() int32 {
		return listIDs[rand.IntN(len(listIDs))]
	}
	randomEntityID := b.makeRandomEntities(mailEntities * b.Factor)
	for i := range mails * b.Factor {
		var mail storage.CreateCharacterMailParams
		isList := spin(0.2)
		if isList {
			recipientIDs := make([]int32, 0)
			m := map[int32]bool{randomListID(): true}
			for range rand.IntN(mailMaxRecipients * b.Factor) {
				m[randomEntityID()] = true
			}
			for id := range m {
				recipientIDs = append(recipientIDs, id)
			}
			mail = storage.CreateCharacterMailParams{
				CharacterID:  b.c.ID,
				FromID:       randomEntityID(),
				RecipientIDs: recipientIDs,
				IsRead:       spin(0.2),
			}
		} else {
			labelID := randomLabelID()
			recipientIDs := make([]int32, 0)
			var fromID int32
			var isRead bool
			switch labelID {
			case app.MailLabelCorp:
				fromID = randomEntityID()
				recipientIDs = append(recipientIDs, b.c.EveCharacter.Corporation.ID)
				isRead = spin(0.2)
			case app.MailLabelAlliance:
				fromID = randomEntityID()
				recipientIDs = append(recipientIDs, b.c.EveCharacter.Alliance.ID)
				isRead = spin(0.2)
			case app.MailLabelSent:
				fromID = b.c.EveCharacter.ID
				recipientIDs = append(recipientIDs, randomEntityID())
			default:
				fromID = randomEntityID()
				m := make(map[int32]bool)
				for range rand.IntN(mailMaxRecipients * b.Factor) {
					m[randomEntityID()] = true
				}
				for id := range m {
					recipientIDs = append(recipientIDs, id)
				}
				isRead = spin(0.2)
			}
			mail = storage.CreateCharacterMailParams{
				CharacterID:  b.c.ID,
				FromID:       fromID,
				RecipientIDs: recipientIDs,
				LabelIDs:     []int32{labelID},
				IsRead:       isRead,
			}
		}
		b.f.CreateCharacterMailWithBody(mail)
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

// spin simulates the spin of a roulette wheel.
// It reports whether an event with the given probability occurred.
func spin(probability float64) bool {
	return rand.Float64() < probability
}
