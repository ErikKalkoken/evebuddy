// Package factory contains factories for creating test objects in the database
package factory

import (
	"context"
	"example/evebuddy/internal/repository"
	"fmt"
	"slices"
	"time"
)

type Factory struct {
	q *repository.Queries
}

func New(q *repository.Queries) Factory {
	f := Factory{q: q}
	return f
}

// CreateCharacter is a test factory for character objects.
func (f Factory) CreateCharacter(args ...repository.CreateCharacterParams) repository.Character {
	ctx := context.Background()
	var arg repository.CreateCharacterParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		ids, err := f.q.ListCharacterIDs(ctx)
		if err != nil {
			panic(err)
		}
		if len(ids) == 0 {
			arg.ID = 1
		} else {
			arg.ID = slices.Max(ids) + 1
		}
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("Generated character #%d", arg.ID)
	}
	if arg.CorporationID == 0 {
		e, err := f.q.CreateEveEntity(ctx, repository.CreateEveEntityParams{Category: repository.EveEntityCorporation})
		if err != nil {
			panic(err)
		}
		arg.CorporationID = e.ID
	}
	if arg.Birthday.IsZero() {
		arg.Birthday = time.Now()
	}
	if arg.Description == "" {
		arg.Description = "Lorem Ipsum"
	}
	// if c.MailUpdatedAt.IsZero() {
	// 	c.MailUpdatedAt = time.Now()
	// }
	character, err := f.q.CreateCharacter(ctx, arg)
	if err != nil {
		panic(err)
	}
	return character
}

// CreateEveEntity is a test factory for EveEntity objects.
func (f Factory) CreateEveEntity(args ...repository.CreateEveEntityParams) repository.EveEntity {
	var arg repository.CreateEveEntityParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		ids, err := f.q.ListEveEntityIDs(ctx)
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
	if arg.Category == "" {
		arg.Category = repository.EveEntityCharacter
	}
	e, err := f.q.CreateEveEntity(ctx, arg)
	if err != nil {
		panic(err)
	}
	return e
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

// // CreateMailLabel is a test factory for MailLabel objects
// func (f factory) CreateMailLabel(args ...repository.MailLabel) repository.MailLabel {
// 	var l repository.MailLabel
// 	if len(args) > 0 {
// 		l = args[0]
// 	}
// 	if l.Character.ID == 0 {
// 		l.Character = f.CreateCharacter()
// 	}
// 	if l.LabelID == 0 {
// 		ll, err := repository.ListMailLabels(l.Character.ID)
// 		if err != nil {
// 			panic(err)
// 		}
// 		var ids []int32
// 		for _, o := range ll {
// 			ids = append(ids, o.LabelID)
// 		}
// 		if len(ids) > 0 {
// 			l.LabelID = slices.Max(ids) + 1
// 		} else {
// 			l.LabelID = 100
// 		}
// 	}
// 	if l.Name == "" {
// 		l.Name = fmt.Sprintf("Generated name #%d", l.LabelID)
// 	}
// 	if l.Color == "" {
// 		l.Color = "#FFFFFF"
// 	}
// 	if l.UnreadCount == 0 {
// 		l.UnreadCount = int32(rand.IntN(1000))
// 	}
// 	if err := l.Save(); err != nil {
// 		panic(err)
// 	}
// 	return l
// }

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

// // CreateToken is a test factory for Token objects.
// func (f factory) CreateToken(args ...repository.Token) repository.Token {
// 	var t repository.Token
// 	if len(args) > 0 {
// 		t = args[0]
// 	}
// 	if t.AccessToken == "" {
// 		t.AccessToken = fmt.Sprintf("GeneratedAccessToken#%d", rand.IntN(1000000))
// 	}
// 	if t.RefreshToken == "" {
// 		t.RefreshToken = fmt.Sprintf("GeneratedRefreshToken#%d", rand.IntN(1000000))
// 	}
// 	if t.ExpiresAt.IsZero() {
// 		t.ExpiresAt = time.Now().Add(time.Minute * 20)
// 	}
// 	if t.TokenType == "" {
// 		t.TokenType = "Bearer"
// 	}
// 	if t.CharacterID == 0 {
// 		c := f.CreateCharacter()
// 		t.CharacterID = c.ID
// 	}
// 	err := t.Save()
// 	if err != nil {
// 		panic(err)
// 	}
// 	return t
// }
