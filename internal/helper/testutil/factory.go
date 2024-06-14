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
	"slices"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

type Factory struct {
	st *storage.Storage
	db *sql.DB
}

func NewFactory(st *storage.Storage, db *sql.DB) Factory {
	f := Factory{st: st, db: db}
	return f
}

func (f Factory) CreateCharacter(args ...storage.UpdateOrCreateCharacterParams) *model.Character {
	ctx := context.Background()
	var arg storage.UpdateOrCreateCharacterParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		c := f.CreateEveCharacter()
		arg.ID = c.ID
	}
	if !arg.HomeID.Valid {
		x := f.CreateLocationStructure()
		arg.HomeID = sql.NullInt64{Int64: x.ID, Valid: true}
	}
	if !arg.LastLoginAt.Valid {
		arg.LastLoginAt = sql.NullTime{Time: time.Now(), Valid: true}
	}
	if !arg.LocationID.Valid {
		x := f.CreateLocationStructure()
		arg.LocationID = sql.NullInt64{Int64: x.ID, Valid: true}
	}
	if !arg.ShipID.Valid {
		x := f.CreateEveType()
		arg.ShipID = sql.NullInt32{Int32: x.ID, Valid: true}
	}
	if !arg.TotalSP.Valid {
		arg.TotalSP = sql.NullInt64{Int64: int64(rand.IntN(100_000_000)), Valid: true}
	}
	if !arg.WalletBalance.Valid {
		arg.WalletBalance = sql.NullFloat64{Float64: rand.Float64() * 100_000_000_000, Valid: true}
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

func (f Factory) CreateCharacterAttributes(args ...storage.UpdateOrCreateCharacterAttributesParams) *model.CharacterAttributes {
	ctx := context.Background()
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

func (f Factory) CreateCharacterAsset(args ...storage.CreateCharacterAssetParams) *model.CharacterAsset {
	ctx := context.Background()
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

func (f Factory) CreateCharacterImplant(args ...storage.CreateCharacterImplantParams) *model.CharacterImplant {
	ctx := context.Background()
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

func (f Factory) CreateCharacterJumpClone(args ...storage.CreateCharacterJumpCloneParams) *model.CharacterJumpClone {
	ctx := context.Background()
	var arg storage.CreateCharacterJumpCloneParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacter()
		arg.CharacterID = x.ID
	}
	if arg.JumpCloneID == 0 {
		arg.JumpCloneID = int64(f.calcNewIDWithCharacter("character_jump_clones", "jump_clone_id", arg.CharacterID))
	}
	if arg.LocationID == 0 {
		x := f.CreateLocationStructure()
		arg.LocationID = x.ID
	}
	if len(arg.Implants) == 0 {
		x := f.CreateEveType()
		arg.Implants = append(arg.Implants, x.ID)
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

func (f Factory) CreateCharacterMail(args ...storage.CreateCharacterMailParams) *model.CharacterMail {
	var arg storage.CreateCharacterMailParams
	ctx := context.Background()
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
		ids, err := f.st.ListCharacterMailIDs(ctx, arg.CharacterID)
		if err != nil {
			panic(err)
		}
		if len(ids) > 0 {
			arg.MailID = slices.Max(ids) + 1
		} else {
			arg.MailID = 1
		}
	}
	if arg.Body == "" {
		arg.Body = fmt.Sprintf("Generated body #%d", arg.MailID)
	}
	if arg.Subject == "" {
		arg.Body = fmt.Sprintf("Generated subject #%d", arg.MailID)
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

func (f Factory) CreateCharacterMailLabel(args ...model.CharacterMailLabel) *model.CharacterMailLabel {
	ctx := context.Background()
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
		ll, err := f.st.ListCharacterMailLabelsOrdered(ctx, arg.CharacterID)
		if err != nil {
			panic(err)
		}
		var ids []int32
		for _, o := range ll {
			ids = append(ids, o.LabelID)
		}
		if len(ids) > 0 {
			arg.LabelID = slices.Max(ids) + 1
		} else {
			arg.LabelID = 100
		}
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("Generated name #%d", arg.LabelID)
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

func (f Factory) CreateCharacterMailList(characterID int32, args ...model.EveEntity) *model.EveEntity {
	var e model.EveEntity
	ctx := context.Background()
	if len(args) > 0 {
		e = args[0]
	}
	if characterID == 0 {
		c := f.CreateCharacter()
		characterID = c.ID
	}
	if e.ID == 0 {
		e = *f.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList})
	}
	if err := f.st.CreateCharacterMailList(ctx, characterID, e.ID); err != nil {
		panic(err)
	}
	return &e
}

func (f Factory) CreateCharacterSkill(args ...storage.UpdateOrCreateCharacterSkillParams) *model.CharacterSkill {
	ctx := context.Background()
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

func (f Factory) CreateCharacterSkillqueueItem(args ...storage.SkillqueueItemParams) *model.CharacterSkillqueueItem {
	ctx := context.Background()
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
	err := f.st.CreateSkillqueueItem(ctx, arg)
	if err != nil {
		panic(err)
	}
	i, err := f.st.GetSkillqueueItem(ctx, arg.CharacterID, arg.QueuePosition)
	if err != nil {
		panic(err)
	}
	return i
}

func (f Factory) CreateCharacterToken(args ...model.CharacterToken) *model.CharacterToken {
	var t model.CharacterToken
	ctx := context.Background()
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
	CharacterID int32
	Section     model.CharacterSection
	Error       string
	CompletedAt time.Time
	StartedAt   time.Time
	Data        any
}

func (f Factory) CreateCharacterSectionStatus(args ...CharacterSectionStatusParams) *model.CharacterSectionStatus {
	ctx := context.Background()
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
	arg2 := storage.CharacterSectionStatusParams{
		CharacterID: arg.CharacterID,
		Section:     arg.Section,
		Error:       arg.Error,
		CompletedAt: arg.CompletedAt,
		ContentHash: hash,
	}
	o, err := f.st.UpdateOrCreateCharacterSectionStatus(ctx, arg2)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterWalletJournalEntry(args ...storage.CreateCharacterWalletJournalEntryParams) *model.CharacterWalletJournalEntry {
	ctx := context.Background()
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
		arg.Amount = rand.Float64() * 10_000_000_000
	}
	if arg.Balance == 0 {
		arg.Amount = rand.Float64() * 100_000_000_000
	}
	if arg.Date.IsZero() {
		arg.Date = time.Now()
	}
	if arg.Description == "" {
		arg.Description = fmt.Sprintf("Description #%d", arg.RefID)
	}
	if arg.Reason == "" {
		arg.Reason = fmt.Sprintf("Reason #%d", arg.RefID)
	}
	if arg.RefType == "" {
		arg.RefType = "player_donation"
	}
	if arg.Tax == 0 {
		arg.Tax = rand.Float64()
	}
	if arg.FirstPartyID == 0 {
		e := f.CreateEveCharacter()
		arg.FirstPartyID = e.ID
	}
	if arg.SecondPartyID == 0 {
		e := f.CreateEveCharacter()
		arg.SecondPartyID = e.ID
	}
	if arg.TaxReceiverID == 0 {
		e := f.CreateEveCharacter()
		arg.TaxReceiverID = e.ID
	}
	err := f.st.CreateCharacterWalletJournalEntry(ctx, arg)
	if err != nil {
		panic(err)
	}
	i, err := f.st.GetCharacterWalletJournalEntry(ctx, arg.CharacterID, arg.RefID)
	if err != nil {
		panic(err)
	}
	return i
}

func (f Factory) CreateCharacterWalletTransaction(args ...storage.CreateCharacterWalletTransactionParams) *model.CharacterWalletTransaction {
	ctx := context.Background()
	var arg storage.CreateCharacterWalletTransactionParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ClientID == 0 {
		x := f.CreateEveCharacter()
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
		arg.TransactionID = int64(f.calcNewIDWithCharacter("character_wallet_transactions", "transaction_id", arg.CharacterID))
	}
	if arg.UnitPrice == 0 {
		arg.UnitPrice = rand.Float64() * 100_000_000
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

func (f Factory) CreateEveCharacter(args ...storage.CreateEveCharacterParams) *model.EveCharacter {
	ctx := context.Background()
	var arg storage.CreateEveCharacterParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_characters", "id", 1))
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("Generated character #%d", arg.ID)
	}
	if arg.CorporationID == 0 {
		c := f.CreateEveEntityCorporation()
		arg.CorporationID = c.ID
	}
	if arg.Birthday.IsZero() {
		arg.Birthday = time.Now()
	}
	if arg.Description == "" {
		arg.Description = "Lorem Ipsum"
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

func (f Factory) CreateEveEntity(args ...model.EveEntity) *model.EveEntity {
	var arg model.EveEntity
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_entities", "id", 1))
	}
	if arg.Category == model.EveEntityUndefined {
		arg.Category = model.EveEntityCharacter
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("%s #%d", arg.Category, arg.ID)
	}
	e, err := f.st.CreateEveEntity(ctx, arg.ID, arg.Name, arg.Category)
	if err != nil {
		panic(fmt.Sprintf("failed to create EveEntity %v: %s", arg, err))
	}
	return e
}

func (f Factory) CreateEveEntityAlliance(args ...model.EveEntity) *model.EveEntity {
	args2 := eveEntityWithCategory(args, model.EveEntityAlliance)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityCharacter(args ...model.EveEntity) *model.EveEntity {
	args2 := eveEntityWithCategory(args, model.EveEntityCharacter)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityCorporation(args ...model.EveEntity) *model.EveEntity {
	args2 := eveEntityWithCategory(args, model.EveEntityCorporation)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntitySolarSystem(args ...model.EveEntity) *model.EveEntity {
	args2 := eveEntityWithCategory(args, model.EveEntitySolarSystem)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityInventoryType(args ...model.EveEntity) *model.EveEntity {
	args2 := eveEntityWithCategory(args, model.EveEntityInventoryType)
	return f.CreateEveEntity(args2...)
}

func eveEntityWithCategory(args []model.EveEntity, category model.EveEntityCategory) []model.EveEntity {
	var e model.EveEntity
	if len(args) > 0 {
		e = args[0]
	}
	e.Category = category
	args2 := []model.EveEntity{e}
	return args2
}
func (f Factory) CreateEveCategory(args ...storage.CreateEveCategoryParams) *model.EveCategory {
	var arg storage.CreateEveCategoryParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_categories", "id", 1))
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("Category #%d", arg.ID)
	}
	r, err := f.st.CreateEveCategory(ctx, arg)
	if err != nil {
		panic(err)
	}
	return r
}

func (f Factory) CreateEveGroup(args ...storage.CreateEveGroupParams) *model.EveGroup {
	var arg storage.CreateEveGroupParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_groups", "id", 1))
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("Group #%d", arg.ID)
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

func (f Factory) CreateEveType(args ...storage.CreateEveTypeParams) *model.EveType {
	var arg storage.CreateEveTypeParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_types", "id", 1))
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
		arg.Name = fmt.Sprintf("Type #%d", arg.ID)
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

func (f Factory) CreateEveTypeDogmaAttribute(args ...storage.CreateEveTypeDogmaAttributeParams) {
	var arg storage.CreateEveTypeDogmaAttributeParams
	ctx := context.Background()
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
}

func (f Factory) CreateEveDogmaAttribute(args ...storage.CreateEveDogmaAttributeParams) *model.EveDogmaAttribute {
	var arg storage.CreateEveDogmaAttributeParams
	ctx := context.Background()
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
		arg.UnitID = model.EveUnitID(rand.IntN(120))
	}
	o, err := f.st.CreateEveDogmaAttribute(ctx, arg)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveRegion(args ...storage.CreateEveRegionParams) *model.EveRegion {
	var arg storage.CreateEveRegionParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_regions", "id", 1))
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("Region #%d", arg.ID)
	}
	o, err := f.st.CreateEveRegion(ctx, arg)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveConstellation(args ...storage.CreateEveConstellationParams) *model.EveConstellation {
	var arg storage.CreateEveConstellationParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_constellations", "id", 1))
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("Constellation #%d", arg.ID)
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

func (f Factory) CreateEveSolarSystem(args ...storage.CreateEveSolarSystemParams) *model.EveSolarSystem {
	var arg storage.CreateEveSolarSystemParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_solar_systems", "id", 1))
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("Solar System #%d", arg.ID)
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

func (f Factory) CreateEveRace(args ...model.EveRace) *model.EveRace {
	var arg model.EveRace
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_races", "id", 1))
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("Race #%d", arg.ID)
	}
	if arg.Description == "" {
		arg.Description = fmt.Sprintf("Description #%d", arg.ID)
	}
	r, err := f.st.CreateEveRace(ctx, arg.ID, arg.Description, arg.Name)
	if err != nil {
		panic(err)
	}
	return r
}

func (f Factory) CreateLocationStructure(args ...storage.UpdateOrCreateLocationParams) *model.EveLocation {
	var arg storage.UpdateOrCreateLocationParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = f.calcNewID("eve_locations", "id", 1_900_000_000_000)
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("Structure #%d", arg.ID)
	}
	if !arg.EveSolarSystemID.Valid {
		x := f.CreateEveSolarSystem()
		arg.EveSolarSystemID = sql.NullInt32{Int32: x.ID, Valid: true}
	}
	if !arg.OwnerID.Valid {
		x := f.CreateEveEntityCorporation()
		arg.OwnerID = sql.NullInt32{Int32: x.ID, Valid: true}
	}
	if !arg.EveTypeID.Valid {
		x := f.CreateEveType()
		arg.EveTypeID = sql.NullInt32{Int32: x.ID, Valid: true}
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

func (f Factory) CreateEveMarketPrice(args ...storage.UpdateOrCreateEveMarketPriceParams) *model.EveMarketPrice {
	var arg storage.UpdateOrCreateEveMarketPriceParams
	ctx := context.Background()
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
	var max sql.NullInt64
	if err := f.db.QueryRow(fmt.Sprintf("SELECT MAX(%s) FROM %s;", id_field, table)).Scan(&max); err != nil {
		panic(err)
	}
	return max.Int64 + start
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
