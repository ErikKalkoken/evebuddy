// Package factory contains factories for creating test objects in the repository
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

type Factory struct {
	r  *storage.Storage
	db *sql.DB
}

func NewFactory(r *storage.Storage, db *sql.DB) Factory {
	f := Factory{r: r, db: db}
	return f
}

// CreateMyCharacter is a test factory for MyCharacter objects.
func (f Factory) CreateMyCharacter(args ...storage.UpdateOrCreateMyCharacterParams) *model.MyCharacter {
	ctx := context.Background()
	var arg storage.UpdateOrCreateMyCharacterParams
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
	err := f.r.UpdateOrCreateMyCharacter(ctx, arg)
	if err != nil {
		panic(err)
	}
	c, err := f.r.GetMyCharacter(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return c
}

// CreateMailLabel is a test factory for MailLabel objects
func (f Factory) CreateMyCharacterUpdateStatus(args ...storage.MyCharacterUpdateStatusParams) *model.MyCharacterUpdateStatus {
	ctx := context.Background()
	var arg storage.MyCharacterUpdateStatusParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.MyCharacterID == 0 {
		c := f.CreateMyCharacter()
		arg.MyCharacterID = c.ID
	}
	if arg.Section == "" {
		panic("missing section")
	}
	if arg.ContentHash == "" {
		arg.ContentHash = fmt.Sprintf("content-hash-%d-%s-%s", arg.MyCharacterID, arg.Section, time.Now())
	}
	if arg.UpdatedAt.IsZero() {
		arg.UpdatedAt = time.Now()
	}
	err := f.r.UpdateOrCreateMyCharacterUpdateStatus(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.r.GetMyCharacterUpdateStatus(ctx, arg.MyCharacterID, arg.Section)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterSkill(args ...storage.UpdateOrCreateCharacterSkillParams) *model.CharacterSkill {
	ctx := context.Background()
	var arg storage.UpdateOrCreateCharacterSkillParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.MyCharacterID == 0 {
		x := f.CreateMyCharacter()
		arg.MyCharacterID = x.ID
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
	err := f.r.UpdateOrCreateCharacterSkill(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.r.GetCharacterSkill(ctx, arg.MyCharacterID, arg.EveTypeID)
	if err != nil {
		panic(err)
	}
	return o
}

// CreateCharacter is a test factory for character objects.
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
	err := f.r.CreateEveCharacter(ctx, arg)
	if err != nil {
		panic(err)
	}
	c, err := f.r.GetEveCharacter(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return c
}

// CreateEveEntity is a test factory for EveEntity objects.
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
	e, err := f.r.CreateEveEntity(ctx, arg.ID, arg.Name, arg.Category)
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

// CreateMail is a test factory for Mail objects
func (f Factory) CreateMail(args ...storage.CreateMailParams) *model.Mail {
	var arg storage.CreateMailParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.MyCharacterID == 0 {
		c := f.CreateMyCharacter()
		arg.MyCharacterID = c.ID
	}
	if arg.FromID == 0 {
		from := f.CreateEveEntityCharacter()
		arg.FromID = from.ID
	}
	if arg.MailID == 0 {
		ids, err := f.r.ListMailIDs(ctx, arg.MyCharacterID)
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
	_, err := f.r.CreateMail(ctx, arg)
	if err != nil {
		panic(err)
	}
	mail, err := f.r.GetMail(ctx, arg.MyCharacterID, arg.MailID)
	if err != nil {
		panic(err)
	}
	return mail
}

// CreateMailLabel is a test factory for MailLabel objects
func (f Factory) CreateMailLabel(args ...model.MailLabel) *model.MailLabel {
	ctx := context.Background()
	var arg storage.MailLabelParams
	if len(args) > 0 {
		l := args[0]
		arg = storage.MailLabelParams{
			MyCharacterID: l.MyCharacterID,
			Color:         l.Color,
			LabelID:       l.LabelID,
			Name:          l.Name,
			UnreadCount:   l.UnreadCount,
		}
	}
	if arg.MyCharacterID == 0 {
		c := f.CreateMyCharacter()
		arg.MyCharacterID = c.ID
	}
	if arg.LabelID == 0 {
		ll, err := f.r.ListMailLabelsOrdered(ctx, arg.MyCharacterID)
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
	label, err := f.r.UpdateOrCreateMailLabel(ctx, arg)
	if err != nil {
		panic(err)
	}
	return label
}

// CreateMailList is a test factory for MailList objects.
func (f Factory) CreateMailList(characterID int32, args ...model.EveEntity) *model.EveEntity {
	var e model.EveEntity
	ctx := context.Background()
	if len(args) > 0 {
		e = args[0]
	}
	if characterID == 0 {
		c := f.CreateMyCharacter()
		characterID = c.ID
	}
	if e.ID == 0 {
		e = *f.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList})
	}
	if err := f.r.CreateMailList(ctx, characterID, e.ID); err != nil {
		panic(err)
	}
	return &e
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
	r, err := f.r.CreateEveCategory(ctx, arg)
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
	err := f.r.CreateEveGroup(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.r.GetEveGroup(ctx, arg.ID)
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
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("Type #%d", arg.ID)
	}
	if arg.GroupID == 0 {
		x := f.CreateEveGroup()
		arg.GroupID = x.ID
	}
	err := f.r.CreateEveType(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.r.GetEveType(ctx, arg.ID)
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
	o, err := f.r.CreateEveRegion(ctx, arg)
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
	err := f.r.CreateEveConstellation(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.r.GetEveConstellation(ctx, arg.ID)
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
		arg.SecurityStatus = rand.Float64()*10 - 5
	}
	err := f.r.CreateEveSolarSystem(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.r.GetEveSolarSystem(ctx, arg.ID)
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
	r, err := f.r.CreateEveRace(ctx, arg.ID, arg.Description, arg.Name)
	if err != nil {
		panic(err)
	}
	return r
}

// CreateSkillqueueItem is a test factory for SkillqueueItem objects
func (f Factory) CreateSkillqueueItem(args ...storage.SkillqueueItemParams) *model.SkillqueueItem {
	ctx := context.Background()
	var arg storage.SkillqueueItemParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	if arg.MyCharacterID == 0 {
		x := f.CreateMyCharacter()
		arg.MyCharacterID = x.ID
	}
	if arg.FinishedLevel == 0 {
		arg.FinishedLevel = rand.IntN(5) + 1
	}
	if arg.LevelEndSP == 0 {
		arg.LevelEndSP = rand.IntN(1_000_000)
	}
	if arg.QueuePosition == 0 {
		var maxPos sql.NullInt64
		q := "SELECT MAX(queue_position) FROM skillqueue_items WHERE my_character_id=?;"
		if err := f.db.QueryRow(q, arg.MyCharacterID).Scan(&maxPos); err != nil {
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
		q2 := "SELECT MAX(finish_date) FROM skillqueue_items WHERE my_character_id=?;"
		if err := f.db.QueryRow(q2, arg.MyCharacterID).Scan(&v); err != nil {
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
	err := f.r.CreateSkillqueueItem(ctx, arg)
	if err != nil {
		panic(err)
	}
	i, err := f.r.GetSkillqueueItem(ctx, arg.MyCharacterID, arg.QueuePosition)
	if err != nil {
		panic(err)
	}
	return i
}

func (f Factory) CreateLocationStructure(args ...storage.UpdateOrCreateLocationParams) *model.Location {
	var arg storage.UpdateOrCreateLocationParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = f.calcNewID("locations", "id", 1_900_000_000_000)
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
	err := f.r.UpdateOrCreateLocation(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.r.GetLocation(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return x
}

// CreateToken is a test factory for Token objects.
func (f Factory) CreateToken(args ...model.Token) *model.Token {
	var t model.Token
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
		c := f.CreateMyCharacter()
		t.CharacterID = c.ID
	}
	err := f.r.UpdateOrCreateToken(ctx, &t)
	if err != nil {
		panic(err)
	}
	return &t
}

func (f Factory) CreateWalletJournalEntry(args ...storage.CreateWalletJournalEntryParams) *model.WalletJournalEntry {
	ctx := context.Background()
	var arg storage.CreateWalletJournalEntryParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.MyCharacterID == 0 {
		x := f.CreateMyCharacter()
		arg.MyCharacterID = x.ID
	}
	if arg.ID == 0 {
		arg.ID = int64(f.calcNewIDWithMyCharacter("wallet_journal_entries", "id", arg.MyCharacterID))
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
		arg.Description = fmt.Sprintf("Description #%d", arg.ID)
	}
	if arg.Reason == "" {
		arg.Reason = fmt.Sprintf("Reason #%d", arg.ID)
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
	err := f.r.CreateWalletJournalEntry(ctx, arg)
	if err != nil {
		panic(err)
	}
	i, err := f.r.GetWalletJournalEntry(ctx, arg.MyCharacterID, arg.ID)
	if err != nil {
		panic(err)
	}
	return i
}

func (f Factory) CreateWalletTransaction(args ...storage.CreateWalletTransactionParams) *model.WalletTransaction {
	ctx := context.Background()
	var arg storage.CreateWalletTransactionParams
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
	if arg.MyCharacterID == 0 {
		x := f.CreateMyCharacter()
		arg.MyCharacterID = x.ID
	}
	if arg.TransactionID == 0 {
		arg.TransactionID = int64(f.calcNewIDWithMyCharacter("wallet_transactions", "transaction_id", arg.MyCharacterID))
	}
	if arg.UnitPrice == 0 {
		arg.UnitPrice = rand.Float64() * 100_000_000
	}

	err := f.r.CreateWalletTransaction(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.r.GetWalletTransaction(ctx, arg.MyCharacterID, arg.TransactionID)
	if err != nil {
		panic(err)
	}
	return x
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

func (f *Factory) calcNewIDWithMyCharacter(table, id_field string, characterID int32) int64 {
	var max sql.NullInt64
	sql := fmt.Sprintf("SELECT MAX(%s) FROM %s WHERE my_character_id = ?;", id_field, table)
	if err := f.db.QueryRow(sql, characterID).Scan(&max); err != nil {
		panic(err)
	}
	return max.Int64 + 1
}
