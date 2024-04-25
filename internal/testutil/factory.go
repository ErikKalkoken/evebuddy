// Package factory contains factories for creating test objects in the repository
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
)

type Factory struct {
	r  *storage.Storage
	db *sql.DB
}

func NewFactory(r *storage.Storage, db *sql.DB) Factory {
	f := Factory{r: r, db: db}
	return f
}

// CreateCharacter is a test factory for character objects.
func (f Factory) CreateCharacter(args ...model.Character) model.Character {
	ctx := context.Background()
	var c model.Character
	if len(args) > 0 {
		c = args[0]
	}
	if c.ID == 0 {
		ids, err := f.r.ListCharacterIDs(ctx)
		if err != nil {
			panic(err)
		}
		if len(ids) == 0 {
			c.ID = 1
		} else {
			c.ID = slices.Max(ids) + 1
		}
	}
	if c.Name == "" {
		c.Name = fmt.Sprintf("Generated character #%d", c.ID)
	}
	if c.Corporation.ID == 0 {
		c.Corporation = f.CreateEveEntityCorporation()
	}
	if c.Birthday.IsZero() {
		c.Birthday = time.Now()
	}
	if c.Description == "" {
		c.Description = "Lorem Ipsum"
	}
	if c.LastLoginAt.IsZero() {
		c.LastLoginAt = time.Now()
	}
	if c.Location.ID == 0 {
		c.Location = f.CreateEveEntitySolarSystem()
	}
	if c.Race.ID == 0 {
		c.Race = f.CreateEveRace()
	}
	if c.Ship.ID == 0 {
		c.Ship = f.CreateEveType()
	}
	if c.SkillPoints == 0 {
		c.SkillPoints = rand.IntN(100_000_000)
	}
	if c.WalletBalance == 0 {
		c.WalletBalance = rand.Float64() * 100_000_000_000
	}
	err := f.r.UpdateOrCreateCharacter(ctx, &c)
	if err != nil {
		panic(err)
	}
	return c
}

// CreateEveEntity is a test factory for EveEntity objects.
func (f Factory) CreateEveEntity(args ...model.EveEntity) model.EveEntity {
	var arg model.EveEntity
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		ids, err := f.r.ListEveEntityIDs(ctx)
		if err != nil {
			panic(err)
		}
		if len(ids) > 0 {
			arg.ID = slices.Max(ids) + 1
		} else {
			arg.ID = 1
		}
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

func (f Factory) CreateEveEntityAlliance(args ...model.EveEntity) model.EveEntity {
	args2 := eveEntityWithCategory(args, model.EveEntityAlliance)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityCharacter(args ...model.EveEntity) model.EveEntity {
	args2 := eveEntityWithCategory(args, model.EveEntityCharacter)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityCorporation(args ...model.EveEntity) model.EveEntity {
	args2 := eveEntityWithCategory(args, model.EveEntityCorporation)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntitySolarSystem(args ...model.EveEntity) model.EveEntity {
	args2 := eveEntityWithCategory(args, model.EveEntitySolarSystem)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityInventoryType(args ...model.EveEntity) model.EveEntity {
	args2 := eveEntityWithCategory(args, model.EveEntityInventoryType)
	return f.CreateEveEntity(args2...)
}

func eveEntityWithCategory(args []model.EveEntity, category model.EveEntityCategory) []model.EveEntity {
	var arg model.EveEntity
	if len(args) > 0 {
		arg = args[0]
	}
	arg.Category = category
	args2 := []model.EveEntity{arg}
	return args2
}

// CreateMail is a test factory for Mail objects
func (f Factory) CreateMail(args ...storage.CreateMailParams) model.Mail {
	var arg storage.CreateMailParams
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
		ids, err := f.r.ListMailIDs(ctx, arg.CharacterID)
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
	mail, err := f.r.GetMail(ctx, arg.CharacterID, arg.MailID)
	if err != nil {
		panic(err)
	}
	return mail
}

// CreateMailLabel is a test factory for MailLabel objects
func (f Factory) CreateMailLabel(args ...model.MailLabel) model.MailLabel {
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
		ll, err := f.r.ListMailLabelsOrdered(ctx, arg.CharacterID)
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
func (f Factory) CreateMailList(characterID int32, args ...model.EveEntity) model.EveEntity {
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
		e = f.CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList})
	}
	if err := f.r.CreateMailList(ctx, characterID, e.ID); err != nil {
		panic(err)
	}
	return e
}

func (f Factory) CreateEveCategory(args ...model.EveCategory) model.EveCategory {
	var x model.EveCategory
	ctx := context.Background()
	if len(args) > 0 {
		x = args[0]
	}
	if x.ID == 0 {
		var max sql.NullInt32
		if err := f.db.QueryRow("SELECT MAX(id) FROM eve_categories;").Scan(&max); err != nil {
			panic(err)
		}
		x.ID = max.Int32 + 1
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

func (f Factory) CreateEveGroup(args ...model.EveGroup) model.EveGroup {
	var x model.EveGroup
	ctx := context.Background()
	if len(args) > 0 {
		x = args[0]
	}
	if x.ID == 0 {
		var max sql.NullInt32
		if err := f.db.QueryRow("SELECT MAX(id) FROM eve_groups;").Scan(&max); err != nil {
			panic(err)
		}
		x.ID = max.Int32 + 1
	}
	if x.Name == "" {
		x.Name = fmt.Sprintf("Group #%d", x.ID)
	}
	if x.Category.ID == 0 {
		x.Category = f.CreateEveCategory()
	}
	err := f.r.CreateEveGroup(ctx, x.ID, x.Category.ID, x.Name, x.IsPublished)
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateEveType(args ...model.EveType) model.EveType {
	var x model.EveType
	ctx := context.Background()
	if len(args) > 0 {
		x = args[0]
	}
	if x.ID == 0 {
		var max sql.NullInt32
		if err := f.db.QueryRow("SELECT MAX(id) FROM eve_types;").Scan(&max); err != nil {
			panic(err)
		}
		x.ID = max.Int32 + 1
	}
	if x.Name == "" {
		x.Name = fmt.Sprintf("Type #%d", x.ID)
	}
	if x.Group.ID == 0 {
		x.Group = f.CreateEveGroup()
	}
	err := f.r.CreateEveType(ctx, x.ID, x.Description, x.Group.ID, x.Name, x.IsPublished)
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateEveRace(args ...model.EveRace) model.EveRace {
	var arg model.EveRace
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		ids, err := f.r.ListEveRaceIDs(ctx)
		if err != nil {
			panic(err)
		}
		if len(ids) > 0 {
			arg.ID = slices.Max(ids) + 1
		} else {
			arg.ID = 1
		}
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

// CreateToken is a test factory for Token objects.
func (f Factory) CreateToken(args ...model.Token) model.Token {
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
		c := f.CreateCharacter()
		t.CharacterID = c.ID
	}
	err := f.r.UpdateOrCreateToken(ctx, &t)
	if err != nil {
		panic(err)
	}
	return t
}
