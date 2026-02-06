package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math/rand/v2"
	"net/http"
	"slices"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

const (
	assetItemsPerLocation = 100
	contracts             = 1
	contractItems         = 1
	implants              = 2
	jumpClones            = 2
	mailLabels            = 10
	marketOrders          = 50
	mailMaxRecipients     = 3
	mails                 = 100
	notifications         = 100
	skillGroups           = 2
	skillqueue            = 2
	skills                = 5
	walletJournalEntries  = 100
	walletTransactions    = 100
)

const (
	corporationDED               = 1000137
	corporationConcord           = 1000125
	corporationStateProtectorate = 1000180
)

type CharacterBuilder struct {
	Factor int

	character     *app.Character
	corporations  []*app.EveEntity
	characterIDs  []int32
	corporationID int32
	eus           *eveuniverseservice.EveUniverseService
	f             *testutil.Factory
	locations     map[int64]*app.EveLocation
	st            *storage.Storage
	types         map[int32]*app.EveType
}

func NewCharacterBuilder(f *testutil.Factory, st *storage.Storage, eus *eveuniverseservice.EveUniverseService, corporationID int32) *CharacterBuilder {
	b := &CharacterBuilder{
		characterIDs:  make([]int32, 0),
		corporationID: corporationID,
		eus:           eus,
		f:             f,
		Factor:        1,
		locations:     make(map[int64]*app.EveLocation),
		st:            st,
		types:         make(map[int32]*app.EveType),
		corporations:  make([]*app.EveEntity, 0),
	}
	return b
}

func (b *CharacterBuilder) Init(ctx context.Context) error {
	if err := b.loadTypes(ctx); err != nil {
		return err
	}
	if err := b.loadLocations(ctx); err != nil {
		return err
	}
	if err := b.loadCharacterIDs(ctx); err != nil {
		return err
	}
	if err := b.loadCorporations(ctx); err != nil {
		return err
	}
	return nil
}

func (b *CharacterBuilder) Create(ctx context.Context) error {
	// must be first
	if err := b.loadRandomCharacter(ctx); err != nil {
		return err
	}
	// any order
	b.createAttributes()
	b.createImplants()
	b.createAssets()
	b.createContracts()
	b.createJumpClones()
	b.createMail()
	if err := b.createMarketOrders(); err != nil {
		return err
	}
	b.createNotifications()
	b.createSkills()
	b.createSkillqueue()
	b.createWalletJournal()
	b.createWalletTransactions()
	// should be last
	b.setCharacterSections()
	b.setGeneralSections()
	return nil
}

func (b *CharacterBuilder) loadCharacterIDs(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://evewho.com/api/corplist/98388312", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "EVE Buddy generate 1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var v eveWhoCorpMembers
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	ids := make([]int32, 0)
	for _, c := range v.Characters {
		ids = append(ids, c.CharacterID)
	}
	rand.Shuffle(len(ids), func(i, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})
	b.characterIDs = ids
	if _, err := b.eus.AddMissingEntities(ctx, set.Of(ids...)); err != nil {
		return err
	}
	return nil
}

func (b *CharacterBuilder) loadCorporations(ctx context.Context) error {
	ids := set.Of(
		b.corporationID,
		corporationDED,
		corporationConcord,
		corporationStateProtectorate,
	)
	ee, err := b.eus.ToEntities(ctx, set.Union(ids))
	if err != nil {
		return err
	}
	b.corporations = slices.Collect(maps.Values(ee))
	return nil
}

func (b *CharacterBuilder) loadTypes(ctx context.Context) error {
	g := new(errgroup.Group)
	g.Go(func() error {
		return b.eus.UpdateCategoryWithChildrenESI(ctx, app.EveCategorySkill)
	})
	g.Go(func() error {
		return b.eus.UpdateCategoryWithChildrenESI(ctx, app.EveCategoryShip)
	})
	if err := g.Wait(); err != nil {
		return err
	}
	if _, err := b.eus.UpdateSectionIfNeeded(ctx, app.GeneralSectionUpdateParams{
		ForceUpdate: true,
		Section:     app.SectionEveMarketPrices,
	}); err != nil {
		return err
	}
	types, err := b.st.ListEveTypes(ctx)
	if err != nil {
		return err
	}
	for _, et := range types {
		// ignore types that may not have a price
		if !et.IsPublished || et.MarketGroupID == 0 || et.Group.ID == 15 {
			continue
		}
		b.types[et.ID] = et
	}
	return nil
}

func (b *CharacterBuilder) randomCharacterID() int32 {
	return sliceRandomElement(b.characterIDs)
}

func (b *CharacterBuilder) randomCorporation() *app.EveEntity {
	return sliceRandomElement(b.corporations)
}

func (b *CharacterBuilder) randomType() *app.EveType {
	s := slices.Collect(maps.Values(b.types))
	return sliceRandomElement(s)
}

func (b *CharacterBuilder) loadLocations(ctx context.Context) error {
	// select itemID from mapDenormalize where groupID = 15 ORDER BY RANDOM() LIMIT 20;
	stationIDs := []int32{
		60011797,
		60001999,
		60008899,
		60000958,
		60000550,
		60000631,
		60014716,
		60006616,
		60014542,
		60013753,
		60012895,
		60007927,
		60003115,
		60001360,
		60014044,
		60006982,
		60012286,
		60000049,
		60015088,
		60000637,
	}
	for _, id := range stationIDs {
		el, err := b.eus.GetOrCreateLocationESI(ctx, int64(id))
		if err != nil {
			return err
		}
		b.locations[el.ID] = el

	}
	return nil
}

func (b *CharacterBuilder) randomLocation() *app.EveLocation {
	s := slices.Collect(maps.Values(b.locations))
	return sliceRandomElement(s)
}

func sliceRandomElement[S ~[]E, E any](s S) E {
	return s[rand.IntN(len(s))]
}

func (b *CharacterBuilder) createAssets() {
	for i, locationID := range xiter.Count(maps.Keys(b.locations), 0) {
		for range assetItemsPerLocation * b.Factor {
			b.f.CreateCharacterAsset(storage.CreateCharacterAssetParams{
				CharacterID: b.character.ID,
				LocationID:  locationID,
				EveTypeID:   b.randomType().ID,
			})
		}
		printProgress("assets", len(b.locations), i)
	}
	printSummary("assets", len(b.locations)*assetItemsPerLocation*b.Factor)
}

func (b *CharacterBuilder) createAttributes() {
	b.f.CreateCharacterAttributes(storage.UpdateOrCreateCharacterAttributesParams{CharacterID: b.character.ID})
	fmt.Println("Created attributes")
}

func (b *CharacterBuilder) loadRandomCharacter(ctx context.Context) error {
	characterID, ok := xslices.Pop(&b.characterIDs)
	if !ok {
		return fmt.Errorf("no character ID found")
	}
	ec, _, err := b.eus.GetOrCreateCharacterESI(ctx, characterID)
	if err != nil {
		return err
	}
	b.character = b.f.CreateCharacterFull(storage.CreateCharacterParams{
		ID: ec.ID,
	})
	b.f.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: characterID})
	fmt.Printf("Creating new character %s with factor %d\n", b.character.EveCharacter.Name, b.Factor)
	return nil
}

type eveWhoCorpMembersCharacter struct {
	CharacterID int32  `json:"character_id"`
	Name        string `json:"name"`
}

type eveWhoCorpMembers struct {
	Characters []eveWhoCorpMembersCharacter `json:"characters"`
}

func (b *CharacterBuilder) createContracts() {
	var count int
	for range contracts * b.Factor {
		o := b.f.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID: b.character.ID,
			Type:        app.ContractTypeItemExchange,
			Price:       float64(rand.IntN(10_000_000*100)) / 100,
		})
		for range contractItems {
			b.f.CreateCharacterContractItem(storage.CreateCharacterContractItemParams{
				ContractID: o.ID,
				TypeID:     b.randomType().ID,
				IsIncluded: true,
			})
		}
		count++
	}
	printSummary("contracts", count)
}

func (b *CharacterBuilder) createMarketOrders() error {
	var count int
	issued := time.Now().Add(-1 * time.Hour)
	ctx := context.Background()
	for i := range marketOrders * b.Factor {
		el := b.randomLocation()
		volumeTotal := rand.IntN(1000) + 10
		volumeRemain := rand.IntN(volumeTotal)
		typeID := b.randomType().ID
		price, err := b.eus.MarketPrice(ctx, typeID)
		if err != nil {
			return err
		}
		b.f.CreateCharacterMarketOrder(storage.UpdateOrCreateCharacterMarketOrderParams{
			CharacterID:   b.character.ID,
			Price:         price.ValueOrFallback(makeRandomPrice()),
			Duration:      rand.IntN(12) + 2,
			IsBuyOrder:    true,
			IsCorporation: false,
			Issued:        issued,
			LocationID:    el.ID,
			OrderID:       int64(i),
			OwnerID:       b.character.ID,
			RegionID:      el.SolarSystem.Constellation.Region.ID,
			State:         app.OrderOpen,
			TypeID:        typeID,
			VolumeRemains: volumeRemain,
			VolumeTotal:   volumeTotal,
		})
		count++
	}
	printSummary("contracts", count)
	return nil
}

func (b *CharacterBuilder) createImplants() {
	for range implants * b.Factor {
		b.f.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: b.character.ID})
	}
	printSummary("jump implants", implants*b.Factor)
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
			CharacterID: b.character.ID,
			Implants:    ii,
		})
	}
	printSummary("jump clones", jumpClones*b.Factor)
}

func (b *CharacterBuilder) createMail() {
	labelIDs := []int32{app.MailLabelInbox, app.MailLabelSent, app.MailLabelCorp, app.MailLabelAlliance}
	for _, l := range labelIDs {
		b.f.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: b.character.ID,
			LabelID:     l,
		})
	}
	for range mailLabels * b.Factor {
		ml := b.f.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: b.character.ID,
		})
		labelIDs = append(labelIDs, ml.LabelID)
	}
	randomLabelID := func() int32 {
		return labelIDs[rand.IntN(len(labelIDs))]
	}
	for i := range mails * b.Factor {
		var mail storage.CreateCharacterMailParams
		isList := spin(0.2)
		var recipientIDs set.Set[int32]
		for range rand.IntN(mailMaxRecipients * b.Factor) {
			recipientIDs.Add(b.randomCharacterID())
		}
		if isList {
			mail = storage.CreateCharacterMailParams{
				CharacterID:  b.character.ID,
				FromID:       b.randomCharacterID(),
				RecipientIDs: slices.Collect(recipientIDs.All()),
				IsRead:       spin(0.2),
			}
		} else {
			labelID := randomLabelID()
			var fromID int32
			var isRead bool
			switch labelID {
			case app.MailLabelCorp:
				fromID = b.randomCharacterID()
				recipientIDs.Add(b.character.EveCharacter.Corporation.ID)
				isRead = spin(0.2)
			case app.MailLabelAlliance:
				id := b.character.EveCharacter.AllianceID()
				if id == 0 {
					continue
				}
				recipientIDs.Add(id)
				fromID = b.randomCharacterID()
				isRead = spin(0.2)
			case app.MailLabelSent:
				fromID = b.character.EveCharacter.ID
				recipientIDs.Add(b.randomCharacterID())
			default:
				fromID = b.randomCharacterID()
				isRead = spin(0.2)
			}
			mail = storage.CreateCharacterMailParams{
				CharacterID:  b.character.ID,
				FromID:       fromID,
				RecipientIDs: slices.Collect(recipientIDs.All()),
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
	for i := range notifications * b.Factor {
		b.f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
			CharacterID: b.character.ID,
			SenderID:    b.randomCorporation().ID,
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
			CharacterID: b.character.ID,
			EveTypeID:   et.ID,
		})
		printProgress("skills", skills*b.Factor, i)
	}
	printSummary("skills", skills*b.Factor)
}

func (b *CharacterBuilder) createSkillqueue() {
	for i := range skillqueue * b.Factor {
		b.f.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: b.character.ID,
		})
		printProgress("skillqueue", skillqueue*b.Factor, i)
	}
	printSummary("skillqueue entries", skillqueue*b.Factor)

}

func (b *CharacterBuilder) createWalletJournal() {
	for i := range walletJournalEntries * b.Factor {
		b.f.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			CharacterID:   b.character.ID,
			FirstPartyID:  b.randomCharacterID(),
			SecondPartyID: b.randomCharacterID(),
			TaxReceiverID: b.randomCharacterID(),
		})
		printProgress("wallet journal", walletJournalEntries*b.Factor, i)
	}
	printSummary("wallet journal entries", walletJournalEntries*b.Factor)
}

func (b *CharacterBuilder) createWalletTransactions() {
	for i := range walletTransactions * b.Factor {
		var isBuy bool
		if rand.Float32() > 0.5 {
			isBuy = true
		}
		b.f.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{
			IsBuy:       isBuy,
			ClientID:    b.randomCharacterID(),
			CharacterID: b.character.ID,
			LocationID:  b.randomLocation().ID,
			EveTypeID:   b.randomType().ID,
		})
		printProgress("wallet transactions", walletTransactions*b.Factor, i)
	}
	printSummary("wallet transactions", walletTransactions*b.Factor)
}

func (b *CharacterBuilder) setCharacterSections() {
	for _, s := range app.CharacterSections {
		b.f.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: b.character.ID,
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

func makeRandomPrice() float64 {
	return float64(rand.IntN(10_000_000*100)) / 100
}
