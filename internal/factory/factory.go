// Package factory contains factories for creating test objects in the repository
package factory

import (
	"context"
	"example/evebuddy/internal/storage"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"
)

type Factory struct {
	r *storage.Repository
}

func New(r *storage.Repository) Factory {
	f := Factory{r: r}
	return f
}

// CreateCharacter is a test factory for character objects.
func (f Factory) CreateCharacter(args ...storage.Character) storage.Character {
	ctx := context.Background()
	var c storage.Character
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
	if c.MailUpdatedAt.IsZero() {
		c.MailUpdatedAt = time.Now()
	}
	err := f.r.UpdateOrCreateCharacter(ctx, &c)
	if err != nil {
		panic(err)
	}
	return c
}

// CreateEveEntity is a test factory for EveEntity objects.
func (f Factory) CreateEveEntity(args ...storage.EveEntity) storage.EveEntity {
	var arg storage.EveEntity
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
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("generated #%d", arg.ID)
	}
	if arg.Category == storage.EveEntityUndefined {
		arg.Category = storage.EveEntityCharacter
	}
	e, err := f.r.CreateEveEntity(ctx, arg.ID, arg.Name, arg.Category)
	if err != nil {
		panic(err)
	}
	return e
}

func (f Factory) CreateEveEntityAlliance(args ...storage.EveEntity) storage.EveEntity {
	args2 := eveEntityWithCategory(args, storage.EveEntityAlliance)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityCharacter(args ...storage.EveEntity) storage.EveEntity {
	args2 := eveEntityWithCategory(args, storage.EveEntityCharacter)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityCorporation(args ...storage.EveEntity) storage.EveEntity {
	args2 := eveEntityWithCategory(args, storage.EveEntityCorporation)
	return f.CreateEveEntity(args2...)
}

func eveEntityWithCategory(args []storage.EveEntity, category storage.EveEntityCategory) []storage.EveEntity {
	var arg storage.EveEntity
	if len(args) > 0 {
		arg = args[0]
	}
	arg.Category = category
	args2 := []storage.EveEntity{arg}
	return args2
}

// CreateMail is a test factory for Mail objects
func (f Factory) CreateMail(args ...storage.CreateMailParams) storage.Mail {
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
func (f Factory) CreateMailLabel(args ...storage.MailLabel) storage.MailLabel {
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
func (f Factory) CreateMailList(characterID int32, args ...storage.EveEntity) storage.EveEntity {
	var e storage.EveEntity
	ctx := context.Background()
	if len(args) > 0 {
		e = args[0]
	}
	if characterID == 0 {
		c := f.CreateCharacter()
		characterID = c.ID
	}
	if e.ID == 0 {
		e = f.CreateEveEntity(storage.EveEntity{Category: storage.EveEntityMailList})
	}
	if err := f.r.CreateMailList(ctx, characterID, e.ID); err != nil {
		panic(err)
	}
	return e
}

// CreateToken is a test factory for Token objects.
func (f Factory) CreateToken(args ...storage.Token) storage.Token {
	var t storage.Token
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
