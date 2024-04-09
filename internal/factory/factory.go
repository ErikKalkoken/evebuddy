// Package factory contains factories for creating test objects in the database
package factory

import (
	"example/evebuddy/internal/model"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"
)

// CreateCharacter is a test factory for character objects.
func CreateCharacter(args ...model.Character) model.Character {
	var c model.Character
	if len(args) > 0 {
		c = args[0]
	}
	if c.ID == 0 {
		ids, err := model.FetchCharacterIDs()
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
		c.Corporation = CreateEveEntity(model.EveEntity{Category: model.EveEntityCorporation})
	}
	// if c.MailUpdatedAt.IsZero() {
	// 	c.MailUpdatedAt = time.Now()
	// }
	err := c.Save()
	if err != nil {
		panic(err)
	}
	return c
}

// CreateEveEntity is a test factory for EveEntity objects.
func CreateEveEntity(args ...model.EveEntity) model.EveEntity {
	var e model.EveEntity
	if len(args) > 0 {
		e = args[0]
	}
	if e.ID == 0 {
		ids, err := model.FetchEveEntityIDs()
		if err != nil {
			panic(err)
		}
		if len(ids) > 0 {
			e.ID = slices.Max(ids) + 1
		} else {
			e.ID = 1
		}
	}
	if e.Name == "" {
		e.Name = fmt.Sprintf("generated #%d", e.ID)
	}
	if e.Category == "" {
		e.Category = model.EveEntityCharacter
	}
	if err := e.Save(); err != nil {
		panic(err)
	}
	return e
}

// CreateMail is a test factory for Mail objects
func CreateMail(args ...model.Mail) model.Mail {
	var m model.Mail
	if len(args) > 0 {
		m = args[0]
	}
	if m.Character.ID == 0 {
		m.Character = CreateCharacter()
	}
	if m.From.ID == 0 {
		m.From = CreateEveEntity(model.EveEntity{Category: model.EveEntityCharacter})
	}
	if m.MailID == 0 {
		ids, err := model.FetchMailIDs(m.Character.ID)
		if err != nil {
			panic(err)
		}
		if len(ids) > 0 {
			m.MailID = slices.Max(ids) + 1
		} else {
			m.MailID = 1
		}
	}
	if m.Body == "" {
		m.Body = fmt.Sprintf("Generated body #%d", m.MailID)
	}
	if m.Subject == "" {
		m.Body = fmt.Sprintf("Generated subject #%d", m.MailID)
	}
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	if err := m.Create(); err != nil {
		panic(err)
	}
	return m
}

// CreateMailLabel is a test factory for MailLabel objects
func CreateMailLabel(args ...model.MailLabel) model.MailLabel {
	var l model.MailLabel
	if len(args) > 0 {
		l = args[0]
	}
	if l.Character.ID == 0 {
		l.Character = CreateCharacter()
	}
	if l.LabelID == 0 {
		ll, err := model.FetchCustomMailLabels(l.Character.ID)
		if err != nil {
			panic(err)
		}
		var ids []int32
		for _, o := range ll {
			ids = append(ids, o.LabelID)
		}
		if len(ids) > 0 {
			l.LabelID = slices.Max(ids) + 1
		} else {
			l.LabelID = 100
		}
	}
	if l.Name == "" {
		l.Name = fmt.Sprintf("Generated name #%d", l.LabelID)
	}
	if l.Color == "" {
		l.Color = "#FFFFFF"
	}
	if l.UnreadCount == 0 {
		l.UnreadCount = int32(rand.IntN(1000))
	}
	if err := l.Save(); err != nil {
		panic(err)
	}
	return l
}

// CreateMailLabel is a test factory for MailList objects.
func CreateMailList(args ...model.MailList) model.MailList {
	var l model.MailList
	if len(args) > 0 {
		l = args[0]
	}
	if l.Character.ID == 0 {
		l.Character = CreateCharacter()
	}
	if l.EveEntity.ID == 0 {
		l.EveEntity = CreateEveEntity(model.EveEntity{Category: model.EveEntityMailList})
	}
	if err := l.CreateIfNew(); err != nil {
		panic(err)
	}
	return l
}

// CreateToken is a test factory for Token objects.
func CreateToken(args ...model.Token) model.Token {
	var t model.Token
	if len(args) > 0 {
		t = args[0]
	}
	if t.Character.ID == 0 || t.CharacterID == 0 {
		t.Character = CreateCharacter()
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
	err := t.Save()
	if err != nil {
		panic(err)
	}
	return t
}
