// Package factory contains factories for creating test objects in the repository
package testutil

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/icrowley/fake"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// EVE IDs
const (
	startIDAlliance      = 99_000_000
	startIDCelestials    = 40_000_000
	startIDCharacter     = 90_000_000
	startIDConstellation = 20_000_000
	startIDCorporation   = 98_000_000
	startIDFaction       = 500_000
	startIDInventoryType = 100
	startIDOther         = 10_000
	startIDRegion        = 10_000_000
	startIDSolarSystem   = 30_000_000
	startIDStation       = 60_000_000
	startIDStructure     = 1_000_000_000_000
)

type Factory struct {
	st *storage.Storage
	db *sql.DB
}

func NewFactory(st *storage.Storage, db *sql.DB) Factory {
	f := Factory{st: st, db: db}
	return f
}

func (f Factory) RandomTime() time.Time {
	hours := time.Duration(rand.IntN(100_000))
	seconds := time.Duration(rand.IntN(3600))
	d := hours*time.Hour + seconds*time.Second
	return time.Now().Add(-d).UTC()
}

func (f Factory) CreateCharacter(args ...storage.UpdateOrCreateCharacterParams) *app.Character {
	ctx := context.TODO()
	var arg storage.UpdateOrCreateCharacterParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		c := f.CreateEveCharacter()
		arg.ID = c.ID
	}
	if arg.HomeID.IsEmpty() {
		x := f.CreateLocationStructure()
		arg.HomeID = optional.New(x.ID)
	}
	if arg.LastLoginAt.IsEmpty() {
		arg.LastLoginAt = optional.New(time.Now())
	}
	if arg.LocationID.IsEmpty() {
		x := f.CreateLocationStructure()
		arg.LocationID = optional.New(x.ID)
	}
	if arg.ShipID.IsEmpty() {
		x := f.CreateEveType()
		arg.ShipID = optional.New(x.ID)
	}
	if arg.TotalSP.IsEmpty() {
		arg.TotalSP = optional.New(rand.IntN(100_000_000))
	}
	if arg.WalletBalance.IsEmpty() {
		arg.WalletBalance = optional.New(rand.Float64() * 100_000_000_000)
	}
	if arg.AssetValue.IsEmpty() {
		arg.AssetValue = optional.New(rand.Float64() * 100_000_000_000)
	}
	err := f.st.UpdateOrCreateCharacter(ctx, arg)
	if err != nil {
		panic(err)
	}
	c, err := f.st.GetCharacter(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return c
}

func (f Factory) CreateCharacterAttributes(args ...storage.UpdateOrCreateCharacterAttributesParams) *app.CharacterAttributes {
	ctx := context.TODO()
	var arg storage.UpdateOrCreateCharacterAttributesParams
	randomValue := func() int {
		return 20 + rand.IntN(5)
	}
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacter()
		arg.CharacterID = x.ID
	}
	if arg.Charisma == 0 {
		arg.Charisma = randomValue()
	}
	if arg.Intelligence == 0 {
		arg.Intelligence = randomValue()
	}
	if arg.Memory == 0 {
		arg.Memory = randomValue()
	}
	if arg.Perception == 0 {
		arg.Perception = randomValue()
	}
	if arg.Willpower == 0 {
		arg.Willpower = randomValue()
	}
	if err := f.st.UpdateOrCreateCharacterAttributes(ctx, arg); err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterAttributes(ctx, arg.CharacterID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterAsset(args ...storage.CreateCharacterAssetParams) *app.CharacterAsset {
	ctx := context.TODO()
	var arg storage.CreateCharacterAssetParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacter()
		arg.CharacterID = x.ID
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	if arg.ItemID == 0 {
		arg.ItemID = f.calcNewIDWithCharacter("character_assets", "item_id", arg.CharacterID)
	}
	if arg.LocationFlag == "" {
		arg.LocationFlag = "Hangar"
	}
	if arg.LocationID == 0 {
		x := f.CreateLocationStructure()
		arg.LocationID = x.ID
	}
	if arg.LocationType == "" {
		arg.LocationType = "other"
	}
	if arg.IsSingleton && arg.Name == "" {
		arg.Name = fmt.Sprintf("Asset %d", arg.ItemID)
	}
	if arg.Quantity == 0 {
		if arg.IsSingleton {
			arg.Quantity = 1
		} else {
			arg.Quantity = rand.Int32N(10_000)
		}
	}
	if err := f.st.CreateCharacterAsset(ctx, arg); err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterAsset(ctx, arg.CharacterID, arg.ItemID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterContract(args ...storage.CreateCharacterContractParams) *app.CharacterContract {
	ctx := context.TODO()
	var arg storage.CreateCharacterContractParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.Availability == "" {
		arg.Availability = "public"
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacter()
		arg.CharacterID = x.ID
	}
	if arg.ContractID == 0 {
		arg.ContractID = int32(f.calcNewIDWithCharacter(
			"character_contracts",
			"contract_id",
			arg.CharacterID,
		))
	}
	if arg.DateIssued.IsZero() {
		arg.DateIssued = time.Now().UTC()
	}
	if arg.DateExpired.IsZero() {
		arg.DateExpired = arg.DateIssued.Add(time.Duration(rand.IntN(200)+12) * time.Hour)
	}
	if arg.IssuerID == 0 {
		c, err := f.st.GetCharacter(ctx, arg.CharacterID)
		if err != nil {
			panic(err)
		}
		_, err = f.st.GetOrCreateEveEntity(ctx, c.ID, c.EveCharacter.Name, app.EveEntityCharacter)
		if err != nil {
			panic(err)
		}
		arg.IssuerID = c.ID
	}
	if arg.IssuerCorporationID == 0 {
		c, err := f.st.GetCharacter(ctx, arg.CharacterID)
		if err != nil {
			panic(err)
		}
		arg.IssuerCorporationID = c.EveCharacter.Corporation.ID
	}
	if arg.Status == "" {
		arg.Status = "outstanding"
	}
	if arg.Type == "" {
		arg.Type = "item_exchange"
	}
	_, err := f.st.CreateCharacterContract(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterContract(ctx, arg.CharacterID, arg.ContractID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterImplant(args ...storage.CreateCharacterImplantParams) *app.CharacterImplant {
	ctx := context.TODO()
	var arg storage.CreateCharacterImplantParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacter()
		arg.CharacterID = x.ID
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	err := f.st.CreateCharacterImplant(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterImplant(ctx, arg.CharacterID, arg.EveTypeID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterJumpClone(args ...storage.CreateCharacterJumpCloneParams) *app.CharacterJumpClone {
	ctx := context.TODO()
	var arg storage.CreateCharacterJumpCloneParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacter()
		arg.CharacterID = x.ID
	}
	if arg.JumpCloneID == 0 {
		arg.JumpCloneID = f.calcNewIDWithCharacter(
			"character_jump_clones",
			"jump_clone_id",
			arg.CharacterID,
		)
	}
	if arg.LocationID == 0 {
		x := f.CreateLocationStructure()
		arg.LocationID = x.ID
	}
	if len(arg.Implants) == 0 {
		x := f.CreateEveType()
		arg.Implants = append(arg.Implants, x.ID)
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("JC-%d", arg.JumpCloneID)
	}
	err := f.st.CreateCharacterJumpClone(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterJumpClone(ctx, arg.CharacterID, int32(arg.JumpCloneID))
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterMail(args ...storage.CreateCharacterMailParams) *app.CharacterMail {
	var arg storage.CreateCharacterMailParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		c := f.CreateCharacter()
		arg.CharacterID = c.ID
	}
	if arg.FromID == 0 {
		from := f.CreateEveEntityCharacter()
		arg.FromID = from.ID
	}
	if arg.MailID == 0 {
		arg.MailID = int32(f.calcNewIDWithCharacter(
			"character_mails",
			"mail_id",
			arg.CharacterID,
		))
	}
	if arg.Body == "" {
		arg.Body = fake.Paragraph()
	}
	if arg.Subject == "" {
		arg.Subject = fake.Sentence()
	}
	if arg.Timestamp.IsZero() {
		arg.Timestamp = time.Now()
	}
	if len(arg.RecipientIDs) == 0 {
		e1 := f.CreateEveEntityCharacter()
		arg.RecipientIDs = []int32{e1.ID}
	}
	_, err := f.st.CreateCharacterMail(ctx, arg)
	if err != nil {
		panic(err)
	}
	mail, err := f.st.GetCharacterMail(ctx, arg.CharacterID, arg.MailID)
	if err != nil {
		panic(err)
	}
	return mail
}

func (f Factory) CreateCharacterMailLabel(args ...app.CharacterMailLabel) *app.CharacterMailLabel {
	ctx := context.TODO()
	var arg storage.MailLabelParams
	if len(args) > 0 {
		l := args[0]
		arg = storage.MailLabelParams{
			CharacterID: l.CharacterID,
			Color:       l.Color,
			LabelID:     l.LabelID,
			Name:        l.Name,
			UnreadCount: l.UnreadCount,
		}
	}
	if arg.CharacterID == 0 {
		c := f.CreateCharacter()
		arg.CharacterID = c.ID
	}
	if arg.LabelID == 0 {
		l := int32(f.calcNewIDWithCharacter("character_mail_labels", "label_id", arg.CharacterID))
		arg.LabelID = max(l, 10) // generate "custom" mail label
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("%s %s", fake.Color(), fake.Language())
	}
	if arg.Color == "" {
		arg.Color = "#FFFFFF"
	}
	if arg.UnreadCount == 0 {
		arg.UnreadCount = int(rand.IntN(1000))
	}
	label, err := f.st.UpdateOrCreateCharacterMailLabel(ctx, arg)
	if err != nil {
		panic(err)
	}
	return label
}

func (f Factory) CreateCharacterMailList(characterID int32, args ...app.EveEntity) *app.EveEntity {
	var e app.EveEntity
	ctx := context.TODO()
	if len(args) > 0 {
		e = args[0]
	}
	if characterID == 0 {
		c := f.CreateCharacter()
		characterID = c.ID
	}
	if e.ID == 0 {
		e = *f.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList})
	}
	if err := f.st.CreateCharacterMailList(ctx, characterID, e.ID); err != nil {
		panic(err)
	}
	return &e
}

func (f Factory) CreateCharacterPlanet(args ...storage.CreateCharacterPlanetParams) *app.CharacterPlanet {
	ctx := context.TODO()
	var arg storage.CreateCharacterPlanetParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacter()
		arg.CharacterID = x.ID
	}
	if arg.EvePlanetID == 0 {
		x := f.CreateEvePlanet()
		arg.EvePlanetID = x.ID
	}
	if arg.UpgradeLevel == 0 {
		arg.UpgradeLevel = rand.IntN(5)
	}
	if arg.LastUpdate.IsZero() {
		arg.LastUpdate = time.Now().UTC()
	}
	_, err := f.st.CreateCharacterPlanet(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterPlanet(ctx, arg.CharacterID, arg.EvePlanetID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreatePlanetPin(args ...storage.CreatePlanetPinParams) *app.PlanetPin {
	ctx := context.TODO()
	var arg storage.CreatePlanetPinParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterPlanetID == 0 {
		x := f.CreateCharacterPlanet()
		arg.CharacterPlanetID = x.ID
	}
	if arg.PinID == 0 {
		arg.PinID = f.calcNewID("planet_pins", "pin_id", 1)
	}
	if arg.TypeID == 0 {
		x := f.CreateEveType()
		arg.TypeID = x.ID
	}
	if err := f.st.CreatePlanetPin(ctx, arg); err != nil {
		panic(err)
	}
	o, err := f.st.GetPlanetPin(ctx, arg.CharacterPlanetID, arg.PinID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterSkill(args ...storage.UpdateOrCreateCharacterSkillParams) *app.CharacterSkill {
	ctx := context.TODO()
	var arg storage.UpdateOrCreateCharacterSkillParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacter()
		arg.CharacterID = x.ID
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	if arg.TrainedSkillLevel == 0 {
		arg.TrainedSkillLevel = rand.IntN(5) + 1
	}
	if arg.ActiveSkillLevel == 0 {
		arg.TrainedSkillLevel = rand.IntN(arg.TrainedSkillLevel) + 1
	}
	if arg.SkillPointsInSkill == 0 {
		arg.SkillPointsInSkill = rand.IntN(1_000_000)
	}
	err := f.st.UpdateOrCreateCharacterSkill(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterSkill(ctx, arg.CharacterID, arg.EveTypeID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterSkillqueueItem(args ...storage.SkillqueueItemParams) *app.CharacterSkillqueueItem {
	ctx := context.TODO()
	var arg storage.SkillqueueItemParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacter()
		arg.CharacterID = x.ID
	}
	if arg.FinishedLevel == 0 {
		arg.FinishedLevel = rand.IntN(5) + 1
	}
	if arg.LevelEndSP == 0 {
		arg.LevelEndSP = rand.IntN(1_000_000)
	}
	if arg.QueuePosition == 0 {
		var maxPos sql.NullInt64
		q := "SELECT MAX(queue_position) FROM character_skillqueue_items WHERE character_id=?;"
		if err := f.db.QueryRow(q, arg.CharacterID).Scan(&maxPos); err != nil {
			panic(err)
		}
		if maxPos.Valid {
			arg.QueuePosition = int(maxPos.Int64) + 1
		} else {
			arg.QueuePosition = int(maxPos.Int64) + 1
		}
	}
	if arg.StartDate.IsZero() {
		var v sql.NullString
		q2 := "SELECT MAX(finish_date) FROM character_skillqueue_items WHERE character_id=?;"
		if err := f.db.QueryRow(q2, arg.CharacterID).Scan(&v); err != nil {
			panic(err)
		}
		if !v.Valid {
			arg.StartDate = time.Now()
		} else {
			maxFinishDate, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", v.String)
			if err != nil {
				panic(err)
			}
			arg.StartDate = maxFinishDate
		}
	}
	if arg.FinishDate.IsZero() {
		hours := rand.IntN(90)*24 + 3
		arg.FinishDate = arg.StartDate.Add(time.Hour * time.Duration(hours))
	}
	err := f.st.CreateCharacterSkillqueueItem(ctx, arg)
	if err != nil {
		panic(err)
	}
	i, err := f.st.GetCharacterSkillqueueItem(ctx, arg.CharacterID, arg.QueuePosition)
	if err != nil {
		panic(err)
	}
	return i
}

func (f Factory) CreateCharacterToken(args ...app.CharacterToken) *app.CharacterToken {
	var t app.CharacterToken
	ctx := context.TODO()
	if len(args) > 0 {
		t = args[0]
	}
	if t.AccessToken == "" {
		t.AccessToken = fmt.Sprintf("GeneratedAccessToken#%d", rand.IntN(1000000))
	}
	if t.RefreshToken == "" {
		t.RefreshToken = fmt.Sprintf("GeneratedRefreshToken#%d", rand.IntN(1000000))
	}
	if t.ExpiresAt.IsZero() {
		t.ExpiresAt = time.Now().Add(time.Minute * 20)
	}
	if t.TokenType == "" {
		t.TokenType = "Bearer"
	}
	if t.CharacterID == 0 {
		c := f.CreateCharacter()
		t.CharacterID = c.ID
	}
	err := f.st.UpdateOrCreateCharacterToken(ctx, &t)
	if err != nil {
		panic(err)
	}
	return &t
}

type CharacterSectionStatusParams struct {
	CharacterID  int32
	Section      app.CharacterSection
	ErrorMessage string
	CompletedAt  time.Time
	StartedAt    time.Time
	Data         any
}

func (f Factory) CreateCharacterSectionStatus(args ...CharacterSectionStatusParams) *app.CharacterSectionStatus {
	ctx := context.TODO()
	var arg CharacterSectionStatusParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		c := f.CreateCharacter()
		arg.CharacterID = c.ID
	}
	if arg.Section == "" {
		panic("must define a section in test factory")
	}
	if arg.Data == "" {
		arg.Data = fmt.Sprintf("content-hash-%d-%s-%s", arg.CharacterID, arg.Section, time.Now())
	}
	if arg.CompletedAt.IsZero() {
		arg.CompletedAt = time.Now()
	}
	if arg.StartedAt.IsZero() {
		arg.StartedAt = time.Now().Add(-1 * time.Duration(rand.IntN(60)) * time.Second)
	}
	hash, err := calcContentHash(arg.Data)
	if err != nil {
		panic(err)
	}
	t := storage.NewNullTimeFromTime(arg.CompletedAt)
	arg2 := storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID:  arg.CharacterID,
		Section:      arg.Section,
		ErrorMessage: &arg.ErrorMessage,
		CompletedAt:  &t,
		ContentHash:  &hash,
	}
	o, err := f.st.UpdateOrCreateCharacterSectionStatus(ctx, arg2)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterWalletJournalEntry(args ...storage.CreateCharacterWalletJournalEntryParams) *app.CharacterWalletJournalEntry {
	ctx := context.TODO()
	var arg storage.CreateCharacterWalletJournalEntryParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacter()
		arg.CharacterID = x.ID
	}
	if arg.RefID == 0 {
		arg.RefID = int64(f.calcNewIDWithCharacter("character_wallet_journal_entries", "id", arg.CharacterID))
	}
	if arg.Amount == 0 {
		var f float64
		if rand.Float32() > 0.5 {
			f = 1
		} else {
			f = -1
		}
		arg.Amount = rand.Float64() * 10_000_000_000 * f
	}
	if arg.Balance == 0 {
		arg.Balance = rand.Float64() * 100_000_000_000
	}
	if arg.Date.IsZero() {
		arg.Date = time.Now()
	}
	if arg.Description == "" {
		arg.Description = fake.Sentence()
	}
	if arg.Reason == "" {
		arg.Reason = fake.Sentence()
	}
	if arg.RefType == "" {
		arg.RefType = "player_donation"
	}
	if arg.Tax == 0 {
		arg.Tax = rand.Float64()
	}
	if arg.FirstPartyID == 0 {
		e := f.CreateEveEntityCharacter()
		arg.FirstPartyID = e.ID
	}
	if arg.SecondPartyID == 0 {
		e := f.CreateEveEntityCharacter()
		arg.SecondPartyID = e.ID
	}
	if arg.TaxReceiverID == 0 {
		e := f.CreateEveEntityCorporation()
		arg.TaxReceiverID = e.ID
	}
	err := f.st.CreateCharacterWalletJournalEntry(ctx, arg)
	if err != nil {
		panic(fmt.Sprintf("%s|%+v", err, arg))
	}
	i, err := f.st.GetCharacterWalletJournalEntry(ctx, arg.CharacterID, arg.RefID)
	if err != nil {
		panic(err)
	}
	return i
}

func (f Factory) CreateCharacterWalletTransaction(args ...storage.CreateCharacterWalletTransactionParams) *app.CharacterWalletTransaction {
	ctx := context.TODO()
	var arg storage.CreateCharacterWalletTransactionParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ClientID == 0 {
		x := f.CreateEveEntityCharacter()
		arg.ClientID = x.ID
	}
	if arg.Date.IsZero() {
		arg.Date = time.Now()
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	if arg.LocationID == 0 {
		x := f.CreateLocationStructure()
		arg.LocationID = x.ID
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacter()
		arg.CharacterID = x.ID
	}
	if arg.TransactionID == 0 {
		arg.TransactionID = f.calcNewIDWithCharacter(
			"character_wallet_transactions",
			"transaction_id",
			arg.CharacterID,
		)
	}
	if arg.UnitPrice == 0 {
		arg.UnitPrice = rand.Float64() * 100_000_000
	}
	if arg.Quantity == 0 {
		arg.Quantity = rand.Int32N(100_000)
	}
	err := f.st.CreateCharacterWalletTransaction(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.st.GetCharacterWalletTransaction(ctx, arg.CharacterID, arg.TransactionID)
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateCharacterNotification(args ...storage.CreateCharacterNotificationParams) *app.CharacterNotification {
	ctx := context.TODO()
	var arg storage.CreateCharacterNotificationParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacter()
		arg.CharacterID = x.ID
	}
	if arg.NotificationID == 0 {
		arg.NotificationID = f.calcNewIDWithCharacter(
			"character_notifications",
			"notification_id",
			arg.CharacterID,
		)
	}
	if arg.SenderID == 0 {
		x := f.CreateEveEntityCorporation()
		arg.SenderID = x.ID
	}
	if arg.Type == "" {
		arg.Type = "CorpBecameWarEligible" // Type without text
	}
	if arg.Timestamp.IsZero() {
		arg.Timestamp = time.Now().UTC()
	}
	err := f.st.CreateCharacterNotification(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.st.GetCharacterNotification(ctx, arg.CharacterID, arg.NotificationID)
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateEveCharacter(args ...storage.CreateEveCharacterParams) *app.EveCharacter {
	ctx := context.TODO()
	var arg storage.CreateEveCharacterParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_characters", "id", startIDCharacter))
	}
	if arg.Name == "" {
		arg.Name = fake.FullName()
	}
	if arg.CorporationID == 0 {
		c := f.CreateEveEntityCorporation()
		arg.CorporationID = c.ID
	}
	if arg.Birthday.IsZero() {
		arg.Birthday = time.Now()
	}
	if arg.Description == "" {
		arg.Description = fake.Paragraphs()
	}
	if arg.RaceID == 0 {
		r := f.CreateEveRace()
		arg.RaceID = r.ID
	}
	err := f.st.CreateEveCharacter(ctx, arg)
	if err != nil {
		panic(err)
	}
	c, err := f.st.GetEveCharacter(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return c
}

type GeneralSectionStatusParams struct {
	Section      app.GeneralSection
	ErrorMessage string
	CompletedAt  time.Time
	StartedAt    time.Time
	Data         any
}

func (f Factory) CreateGeneralSectionStatus(args ...GeneralSectionStatusParams) *app.GeneralSectionStatus {
	ctx := context.TODO()
	var arg GeneralSectionStatusParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.Section == "" {
		panic("must define a section in test factory")
	}
	if arg.Data == "" {
		arg.Data = fmt.Sprintf("content-hash-%s-%s", arg.Section, time.Now())
	}
	if arg.CompletedAt.IsZero() {
		arg.CompletedAt = time.Now()
	}
	if arg.StartedAt.IsZero() {
		arg.StartedAt = time.Now().Add(-1 * time.Duration(rand.IntN(60)) * time.Second)
	}
	hash, err := calcContentHash(arg.Data)
	if err != nil {
		panic(err)
	}
	t := storage.NewNullTimeFromTime(arg.CompletedAt)
	arg2 := storage.UpdateOrCreateGeneralSectionStatusParams{
		Section:     arg.Section,
		Error:       &arg.ErrorMessage,
		CompletedAt: &t,
		ContentHash: &hash,
	}
	o, err := f.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveEntity(args ...app.EveEntity) *app.EveEntity {
	var arg app.EveEntity
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.Category == app.EveEntityUndefined {
		arg.Category = app.EveEntityCharacter
	}
	if arg.ID == 0 {
		var start int64
		m := map[app.EveEntityCategory]int64{
			app.EveEntityAlliance:      startIDAlliance,
			app.EveEntityCharacter:     startIDCharacter,
			app.EveEntityCorporation:   startIDCorporation,
			app.EveEntityFaction:       startIDFaction,
			app.EveEntityInventoryType: startIDInventoryType,
			app.EveEntitySolarSystem:   startIDSolarSystem,
			app.EveEntityStation:       startIDStation,
		}
		start, ok := m[arg.Category]
		if !ok {
			start = startIDOther
		}
		arg.ID = int32(f.calcNewID("eve_entities", "id", start))
	}
	if arg.Name == "" {
		switch arg.Category {
		case app.EveEntityCharacter:
			arg.Name = fake.FullName()
		case app.EveEntityCorporation:
			arg.Name = fake.Company()
		case app.EveEntityAlliance:
			arg.Name = fake.Company()
		case app.EveEntityFaction:
			arg.Name = fake.JobTitle()
		case app.EveEntityMailList:
			arg.Name = fmt.Sprintf("%s %s", fake.Color(), fake.Industry())
		default:
			arg.Name = fmt.Sprintf("%s #%d", arg.Category, arg.ID)
		}
	}
	e, err := f.st.CreateEveEntity(ctx, arg.ID, arg.Name, arg.Category)
	if err != nil {
		panic(fmt.Sprintf("create EveEntity %v: %s", arg, err))
	}
	return e
}

func (f Factory) CreateEveEntityAlliance(args ...app.EveEntity) *app.EveEntity {
	args2 := eveEntityWithCategory(args, app.EveEntityAlliance)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityCharacter(args ...app.EveEntity) *app.EveEntity {
	args2 := eveEntityWithCategory(args, app.EveEntityCharacter)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityCorporation(args ...app.EveEntity) *app.EveEntity {
	args2 := eveEntityWithCategory(args, app.EveEntityCorporation)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntitySolarSystem(args ...app.EveEntity) *app.EveEntity {
	args2 := eveEntityWithCategory(args, app.EveEntitySolarSystem)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityInventoryType(args ...app.EveEntity) *app.EveEntity {
	args2 := eveEntityWithCategory(args, app.EveEntityInventoryType)
	return f.CreateEveEntity(args2...)
}

func eveEntityWithCategory(args []app.EveEntity, category app.EveEntityCategory) []app.EveEntity {
	var e app.EveEntity
	if len(args) > 0 {
		e = args[0]
	}
	e.Category = category
	args2 := []app.EveEntity{e}
	return args2
}

func (f Factory) CreateEveCategory(args ...storage.CreateEveCategoryParams) *app.EveCategory {
	var arg storage.CreateEveCategoryParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_categories", "id", 1))
	}
	if arg.Name == "" {
		arg.Name = fake.Industry()
	}
	r, err := f.st.CreateEveCategory(ctx, arg)
	if err != nil {
		panic(err)
	}
	return r
}

func (f Factory) CreateEveGroup(args ...storage.CreateEveGroupParams) *app.EveGroup {
	var arg storage.CreateEveGroupParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_groups", "id", 1))
	}
	if arg.Name == "" {
		arg.Name = fake.Brand()
	}
	if arg.CategoryID == 0 {
		x := f.CreateEveCategory()
		arg.CategoryID = x.ID
	}
	err := f.st.CreateEveGroup(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetEveGroup(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveType(args ...storage.CreateEveTypeParams) *app.EveType {
	var arg storage.CreateEveTypeParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_types", "id", startIDInventoryType))
	}
	if arg.GroupID == 0 {
		x := f.CreateEveGroup()
		arg.GroupID = x.ID
	}
	if arg.Capacity == 0 {
		arg.Capacity = rand.Float32() * 1_000_000
	}
	if arg.Mass == 0 {
		arg.Mass = rand.Float32() * 10_000_000_000
	}
	if arg.Name == "" {
		arg.Name = fake.ProductName()
	}
	if arg.Description == "" {
		arg.Description = fake.Paragraph()
	}
	if arg.PortionSize == 0 {
		arg.PortionSize = 1
	}
	if arg.Radius == 0 {
		arg.Radius = rand.Float32() * 10_000
	}
	if arg.Volume == 0 {
		arg.Volume = rand.Float32() * 10_000_000
	}
	err := f.st.CreateEveType(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetEveType(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveTypeDogmaAttribute(args ...storage.CreateEveTypeDogmaAttributeParams) *app.EveTypeDogmaAttribute {
	var arg storage.CreateEveTypeDogmaAttributeParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	if arg.DogmaAttributeID == 0 {
		x := f.CreateEveDogmaAttribute()
		arg.DogmaAttributeID = x.ID
	}
	if arg.Value == 0 {
		arg.Value = rand.Float32() * 10_000
	}
	if err := f.st.CreateEveTypeDogmaAttribute(ctx, arg); err != nil {
		panic(err)
	}
	v, err := f.st.GetEveTypeDogmaAttribute(ctx, arg.EveTypeID, arg.DogmaAttributeID)
	if err != nil {
		panic(err)
	}
	et, err := f.st.GetEveType(ctx, arg.EveTypeID)
	if err != nil {
		panic(err)
	}
	da, err := f.st.GetEveDogmaAttribute(ctx, arg.DogmaAttributeID)
	if err != nil {
		panic(err)
	}
	o := &app.EveTypeDogmaAttribute{
		EveType:        et,
		DogmaAttribute: da,
		Value:          v,
	}
	return o
}

func (f Factory) CreateEveDogmaAttribute(args ...storage.CreateEveDogmaAttributeParams) *app.EveDogmaAttribute {
	var arg storage.CreateEveDogmaAttributeParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_dogma_attributes", "id", 1))
	}
	if arg.DefaultValue == 0 {
		arg.DefaultValue = rand.Float32() * 10_000
	}
	if arg.Description == "" {
		arg.Description = fmt.Sprintf("Description #%d", arg.ID)
	}
	if arg.DisplayName == "" {
		arg.DisplayName = fmt.Sprintf("Display Name #%d", arg.ID)
	}
	if arg.IconID == 0 {
		arg.IconID = rand.Int32N(100_000)
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("Name #%d", arg.ID)
	}
	if arg.UnitID == 0 {
		arg.UnitID = app.EveUnitID(rand.IntN(120))
	}
	o, err := f.st.CreateEveDogmaAttribute(ctx, arg)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveRegion(args ...storage.CreateEveRegionParams) *app.EveRegion {
	var arg storage.CreateEveRegionParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_regions", "id", startIDRegion))
	}
	if arg.Name == "" {
		arg.Name = fake.Continent()
	}
	o, err := f.st.CreateEveRegion(ctx, arg)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveConstellation(args ...storage.CreateEveConstellationParams) *app.EveConstellation {
	var arg storage.CreateEveConstellationParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_constellations", "id", startIDConstellation))
	}
	if arg.Name == "" {
		arg.Name = fake.Country()
	}
	if arg.RegionID == 0 {
		x := f.CreateEveRegion()
		arg.RegionID = x.ID
	}
	err := f.st.CreateEveConstellation(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetEveConstellation(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveSolarSystem(args ...storage.CreateEveSolarSystemParams) *app.EveSolarSystem {
	var arg storage.CreateEveSolarSystemParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_solar_systems", "id", startIDSolarSystem))
	}
	if arg.Name == "" {
		arg.Name = fake.City()
	}
	if arg.ConstellationID == 0 {
		x := f.CreateEveConstellation()
		arg.ConstellationID = x.ID
	}
	if arg.SecurityStatus == 0 {
		arg.SecurityStatus = rand.Float32()*10 - 5
	}
	err := f.st.CreateEveSolarSystem(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetEveSolarSystem(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEvePlanet(args ...storage.CreateEvePlanetParams) *app.EvePlanet {
	var arg storage.CreateEvePlanetParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_planets", "id", startIDCelestials))
	}
	if arg.Name == "" {
		arg.Name = fake.Street()
	}
	if arg.SolarSystemID == 0 {
		x := f.CreateEveSolarSystem()
		arg.SolarSystemID = x.ID
	}
	if arg.TypeID == 0 {
		x := f.CreateEveType()
		arg.TypeID = x.ID
	}
	err := f.st.CreateEvePlanet(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetEvePlanet(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveMoon(args ...storage.CreateEveMoonParams) *app.EveMoon {
	var arg storage.CreateEveMoonParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_moons", "id", startIDCelestials))
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("%s %s", fake.Color(), fake.Street())
	}
	if arg.SolarSystemID == 0 {
		x := f.CreateEveSolarSystem()
		arg.SolarSystemID = x.ID
	}
	err := f.st.CreateEveMoon(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetEveMoon(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveRace(args ...app.EveRace) *app.EveRace {
	var arg app.EveRace
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_races", "id", startIDOther))
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("%s #%d", fake.JobTitle(), arg.ID)
	}
	if arg.Description == "" {
		arg.Description = fake.Paragraph()
	}
	r, err := f.st.CreateEveRace(ctx, arg.ID, arg.Description, arg.Name)
	if err != nil {
		panic(err)
	}
	return r
}

func (f Factory) CreateEveSchematic(args ...storage.CreateEveSchematicParams) *app.EveSchematic {
	var arg storage.CreateEveSchematicParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_schematics", "id", 1))
	}
	if arg.Name == "" {
		arg.Name = fake.ProductName()
	}
	r, err := f.st.CreateEveSchematic(ctx, arg)
	if err != nil {
		panic(err)
	}
	return r
}

func (f Factory) CreateLocationStructure(args ...storage.UpdateOrCreateLocationParams) *app.EveLocation {
	var arg storage.UpdateOrCreateLocationParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = f.calcNewID("eve_locations", "id", startIDStructure)
	}
	if arg.Name == "" {
		arg.Name = fake.Color() + " " + fake.Brand()
	}
	if arg.EveSolarSystemID.IsEmpty() {
		x := f.CreateEveSolarSystem()
		arg.EveSolarSystemID = optional.New(x.ID)
	}
	if arg.OwnerID.IsEmpty() {
		x := f.CreateEveEntityCorporation()
		arg.OwnerID = optional.New(x.ID)
	}
	if arg.EveTypeID.IsEmpty() {
		x := f.CreateEveType()
		arg.EveTypeID = optional.New(x.ID)
	}
	if arg.UpdatedAt.IsZero() {
		arg.UpdatedAt = time.Now()
	}
	err := f.st.UpdateOrCreateEveLocation(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.st.GetEveLocation(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateEveMarketPrice(args ...storage.UpdateOrCreateEveMarketPriceParams) *app.EveMarketPrice {
	var arg storage.UpdateOrCreateEveMarketPriceParams
	ctx := context.TODO()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.TypeID == 0 {
		arg.TypeID = int32(f.calcNewID("eve_market_price", "type_id", 1))
	}
	if arg.AdjustedPrice == 0 {
		arg.AdjustedPrice = rand.Float64() * 100_000
	}
	if arg.AveragePrice == 0 {
		arg.AveragePrice = rand.Float64() * 100_000
	}
	err := f.st.UpdateOrCreateEveMarketPrice(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetEveMarketPrice(ctx, arg.TypeID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f *Factory) calcNewID(table, id_field string, start int64) int64 {
	if start < 1 {
		panic("start must be a positive number")
	}
	var vMax sql.NullInt64
	if err := f.db.QueryRow(fmt.Sprintf("SELECT MAX(%s) FROM %s;", id_field, table)).Scan(&vMax); err != nil {
		panic(err)
	}
	return max(vMax.Int64+1, start)
}

func (f *Factory) calcNewIDWithCharacter(table, id_field string, characterID int32) int64 {
	var max sql.NullInt64
	sql := fmt.Sprintf("SELECT MAX(%s) FROM %s WHERE character_id = ?;", id_field, table)
	if err := f.db.QueryRow(sql, characterID).Scan(&max); err != nil {
		panic(err)
	}
	return max.Int64 + 1
}

// func (f *Factory) calcNewIDWithParam(table, id_field, where_field string, where_value int64) int64 {
// 	var max sql.NullInt64
// 	sql := fmt.Sprintf("SELECT MAX(%s) FROM %s WHERE %s = ?;", id_field, table, where_field)
// 	if err := f.db.QueryRow(sql, where_value).Scan(&max); err != nil {
// 		panic(err)
// 	}
// 	return max.Int64 + 1
// }

func calcContentHash(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	b2 := md5.Sum(b)
	hash := hex.EncodeToString(b2[:])
	return hash, nil
}
