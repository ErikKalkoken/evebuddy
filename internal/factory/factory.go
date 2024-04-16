// Package factory contains factories for creating test objects in the repository
package factory

import (
	"context"
	"example/evebuddy/internal/repository"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"
)

type Factory struct {
	r *repository.Repository
}

func New(r *repository.Repository) Factory {
	f := Factory{r: r}
	return f
}

// CreateCharacter is a test factory for character objects.
func (f Factory) CreateCharacter(args ...repository.Character) repository.Character {
	ctx := context.Background()
	var c repository.Character
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
	// if c.MailUpdatedAt.IsZero() {
	// 	c.MailUpdatedAt = time.Now()
	// }
	err := f.r.UpdateOrCreateCharacter(ctx, &c)
	if err != nil {
		panic(err)
	}
	return c
}

// CreateEveEntity is a test factory for EveEntity objects.
func (f Factory) CreateEveEntity(args ...repository.EveEntity) repository.EveEntity {
	var arg repository.EveEntity
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
	if arg.Category == repository.EveEntityUndefined {
		arg.Category = repository.EveEntityCharacter
	}
	e, err := f.r.CreateEveEntity(ctx, arg.ID, arg.Name, arg.Category)
	if err != nil {
		panic(err)
	}
	return e
}

func (f Factory) CreateEveEntityAlliance(args ...repository.EveEntity) repository.EveEntity {
	args2 := eveEntityWithCategory(args, repository.EveEntityAlliance)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityCharacter(args ...repository.EveEntity) repository.EveEntity {
	args2 := eveEntityWithCategory(args, repository.EveEntityCharacter)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityCorporation(args ...repository.EveEntity) repository.EveEntity {
	args2 := eveEntityWithCategory(args, repository.EveEntityCorporation)
	return f.CreateEveEntity(args2...)
}

func eveEntityWithCategory(args []repository.EveEntity, category repository.EveEntityCategory) []repository.EveEntity {
	var arg repository.EveEntity
	if len(args) > 0 {
		arg = args[0]
	}
	arg.Category = category
	args2 := []repository.EveEntity{arg}
	return args2
}

// // CreateMail is a test factory for Mail objects
// func (f factory) CreateMail(args ...repository.Mail) repository.Mail {
// 	var m repository.Mail
// 	if len(args) > 0 {
// 		m = args[0]
// 	}
// 	if m.Character.ID == 0 {
// 		m.Character = f.CreateCharacter()
// 	}
// 	if m.From.ID == 0 {
// 		m.From = f.CreateEveEntity(repository.EveEntity{Category: repository.EveEntityCharacter})
// 	}
// 	if m.MailID == 0 {
// 		ids, err := repository.ListMailIDs(m.Character.ID)
// 		if err != nil {
// 			panic(err)
// 		}
// 		if len(ids) > 0 {
// 			m.MailID = slices.Max(ids) + 1
// 		} else {
// 			m.MailID = 1
// 		}
// 	}
// 	if m.Body == "" {
// 		m.Body = fmt.Sprintf("Generated body #%d", m.MailID)
// 	}
// 	if m.Subject == "" {
// 		m.Body = fmt.Sprintf("Generated subject #%d", m.MailID)
// 	}
// 	if m.Timestamp.IsZero() {
// 		m.Timestamp = time.Now()
// 	}
// 	if err := m.Create(); err != nil {
// 		panic(err)
// 	}
// 	return m
// }

// CreateMailLabel is a test factory for MailLabel objects
func (f Factory) CreateMailLabel(args ...repository.MailLabel) repository.MailLabel {
	ctx := context.Background()
	var arg repository.MailLabelParams
	if len(args) > 0 {
		l := args[0]
		arg = repository.MailLabelParams{
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
		ll, err := f.r.ListMailLabels(ctx, arg.CharacterID)
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

// // CreateMailLabel is a test factory for MailList objects.
// func (f factory) CreateMailList(args ...repository.MailList) repository.MailList {
// 	var l repository.MailList
// 	if len(args) > 0 {
// 		l = args[0]
// 	}
// 	if l.Character.ID == 0 {
// 		l.Character = f.CreateCharacter()
// 	}
// 	if l.EveEntity.ID == 0 {
// 		l.EveEntity = f.CreateEveEntity(repository.EveEntity{Category: repository.EveEntityMailList})
// 	}
// 	if err := l.CreateIfNew(); err != nil {
// 		panic(err)
// 	}
// 	return l
// }

// CreateToken is a test factory for Token objects.
func (f Factory) CreateToken(args ...repository.Token) repository.Token {
	var t repository.Token
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
