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
func (f Factory) CreateMyCharacter(args ...model.MyCharacter) *model.MyCharacter {
	ctx := context.Background()
	var c model.MyCharacter
	if len(args) > 0 {
		c = args[0]
	}
	if c.ID == 0 {
		c.ID = int32(f.calcNewID("my_characters", "id"))
	}
	if c.Character == nil {
		c.Character = f.CreateEveCharacter()
	}
	if c.LastLoginAt.IsZero() {
		c.LastLoginAt = time.Now()
	}
	if c.Location == nil {
		c.Location = f.CreateEveSolarSystem()
	}
	if c.Ship == nil {
		c.Ship = f.CreateEveType()
	}
	if c.SkillPoints == 0 {
		c.SkillPoints = rand.IntN(100_000_000)
	}
	if c.WalletBalance == 0 {
		c.WalletBalance = rand.Float64() * 100_000_000_000
	}
	err := f.r.UpdateOrCreateMyCharacter(ctx, &c)
	if err != nil {
		panic(err)
	}
	return &c
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

// CreateCharacter is a test factory for character objects.
func (f Factory) CreateEveCharacter(args ...storage.CreateEveCharacterParams) *model.EveCharacter {
	ctx := context.Background()
	var arg storage.CreateEveCharacterParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_characters", "id"))
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
		arg.ID = int32(f.calcNewID("eve_entities", "id"))
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

func (f Factory) CreateEveCategory(args ...model.EveCategory) *model.EveCategory {
	var x model.EveCategory
	ctx := context.Background()
	if len(args) > 0 {
		x = args[0]
	}
	if x.ID == 0 {
		x.ID = int32(f.calcNewID("eve_categories", "id"))
	}
	if x.Name == "" {
		x.Name = fmt.Sprintf("Category #%d", x.ID)
	}
	r, err := f.r.CreateEveCategory(ctx, x.ID, x.Name, x.IsPublished)
	if err != nil {
		panic(err)
	}
	return r
}

func (f Factory) CreateEveGroup(args ...model.EveGroup) *model.EveGroup {
	var x model.EveGroup
	ctx := context.Background()
	if len(args) > 0 {
		x = args[0]
	}
	if x.ID == 0 {
		x.ID = int32(f.calcNewID("eve_groups", "id"))
	}
	if x.Name == "" {
		x.Name = fmt.Sprintf("Group #%d", x.ID)
	}
	if x.Category == nil {
		x.Category = f.CreateEveCategory()
	}
	err := f.r.CreateEveGroup(ctx, x.ID, x.Category.ID, x.Name, x.IsPublished)
	if err != nil {
		panic(err)
	}
	return &x
}

func (f Factory) CreateEveType(args ...model.EveType) *model.EveType {
	var x model.EveType
	ctx := context.Background()
	if len(args) > 0 {
		x = args[0]
	}
	if x.ID == 0 {
		x.ID = int32(f.calcNewID("eve_types", "id"))
	}
	if x.Name == "" {
		x.Name = fmt.Sprintf("Type #%d", x.ID)
	}
	if x.Group == nil {
		x.Group = f.CreateEveGroup()
	}
	err := f.r.CreateEveType(ctx, x.ID, x.Description, x.Group.ID, x.Name, x.IsPublished)
	if err != nil {
		panic(err)
	}
	return &x
}

func (f Factory) CreateEveRegion(args ...model.EveRegion) *model.EveRegion {
	var x model.EveRegion
	ctx := context.Background()
	if len(args) > 0 {
		x = args[0]
	}
	if x.ID == 0 {
		x.ID = int32(f.calcNewID("eve_regions", "id"))
	}
	if x.Name == "" {
		x.Name = fmt.Sprintf("Region #%d", x.ID)
	}
	r, err := f.r.CreateEveRegion(ctx, x.Description, x.ID, x.Name)
	if err != nil {
		panic(err)
	}
	return r
}

func (f Factory) CreateEveConstellation(args ...model.EveConstellation) *model.EveConstellation {
	var x model.EveConstellation
	ctx := context.Background()
	if len(args) > 0 {
		x = args[0]
	}
	if x.ID == 0 {
		x.ID = int32(f.calcNewID("eve_constellations", "id"))
	}
	if x.Name == "" {
		x.Name = fmt.Sprintf("Constellation #%d", x.ID)
	}
	if x.Region == nil {
		x.Region = f.CreateEveRegion()
	}
	err := f.r.CreateEveConstellation(ctx, x.ID, x.Region.ID, x.Name)
	if err != nil {
		panic(err)
	}
	return &x
}

func (f Factory) CreateEveSolarSystem(args ...model.EveSolarSystem) *model.EveSolarSystem {
	var x model.EveSolarSystem
	ctx := context.Background()
	if len(args) > 0 {
		x = args[0]
	}
	if x.ID == 0 {
		x.ID = int32(f.calcNewID("eve_solar_systems", "id"))
	}
	if x.Name == "" {
		x.Name = fmt.Sprintf("Solar System #%d", x.ID)
	}
	if x.Constellation == nil {
		x.Constellation = f.CreateEveConstellation()
	}
	if x.SecurityStatus == 0 {
		x.SecurityStatus = rand.Float64()*10 - 5
	}
	err := f.r.CreateEveSolarSystem(ctx, x.ID, x.Constellation.ID, x.Name, x.SecurityStatus)
	if err != nil {
		panic(err)
	}
	return &x
}

func (f Factory) CreateEveRace(args ...model.EveRace) *model.EveRace {
	var arg model.EveRace
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_races", "id"))
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

// CreateWalletJournalEntry is a test factory for WalletJournalEntry objects
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
		arg.ID = int64(f.calcNewID("wallet_journal_entries", "id"))
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
func (f *Factory) calcNewID(table, id_field string) int {
	var max sql.NullInt64
	if err := f.db.QueryRow(fmt.Sprintf("SELECT MAX(%s) FROM %s;", id_field, table)).Scan(&max); err != nil {
		panic(err)
	}
	return int(max.Int64 + 1)
}
