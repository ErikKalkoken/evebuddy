package model_test

import (
	"example/esiapp/internal/helper/set"
	"example/esiapp/internal/model"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createMail(args ...model.Mail) model.Mail {
	var m model.Mail
	if len(args) > 0 {
		m = args[0]
	}
	if m.Character.ID == 0 {
		m.Character = createCharacter()
	}
	if m.From.ID == 0 {
		m.From = createEveEntity(model.EveEntity{Category: model.EveEntityCharacter})
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

func TestMailCanCreateNew(t *testing.T) {
	// given
	model.TruncateTables()
	c := createCharacter()
	f := createEveEntity()
	r := createEveEntity()
	l := createMailLabel(model.MailLabel{Character: c})
	m := model.Mail{
		Body:       "body",
		Character:  c,
		From:       f,
		Labels:     []model.MailLabel{l},
		MailID:     7,
		Recipients: []model.EveEntity{r},
		Subject:    "subject",
		Timestamp:  time.Now(),
	}
	// when
	err := m.Create()
	// then
	assert.NoError(t, err)
	m2, err := model.FetchMail(m.ID)
	assert.NoError(t, err)
	assert.Equal(t, m.MailID, m2.MailID)
	assert.Equal(t, m.Body, m2.Body)
	assert.Equal(t, f.ID, m2.FromID)
	assert.Equal(t, f, m2.From)
	assert.Equal(t, c.ID, m2.CharacterID)
	assert.Equal(t, m.Subject, m2.Subject)
	assert.Equal(t, m.Timestamp.Unix(), m2.Timestamp.Unix())
	assert.Equal(t, []model.EveEntity{r}, m2.Recipients)
	assert.Equal(t, l.Name, m2.Labels[0].Name)
	assert.Equal(t, l.LabelID, m2.Labels[0].LabelID)
}

func TestMailCreateShouldReturnErrorWhenCharacterIDMissing(t *testing.T) {
	// given
	model.TruncateTables()
	from := createEveEntity()
	m := model.Mail{
		Body:      "body",
		From:      from,
		MailID:    7,
		Subject:   "subject",
		Timestamp: time.Now(),
	}
	// when
	err := m.Create()
	// then
	assert.Error(t, err)
}

func TestMailCreateShouldReturnErrorWhenFromIDMissing(t *testing.T) {
	// given
	model.TruncateTables()
	c := createCharacter()
	m := model.Mail{
		Body:      "body",
		Character: c,
		MailID:    7,
		Subject:   "subject",
		Timestamp: time.Now(),
	}
	// when
	err := m.Create()
	// then
	assert.Error(t, err)
}

func TestMailCanFetchMailIDs(t *testing.T) {
	// given
	model.TruncateTables()
	char := createCharacter()
	for i := range 3 {
		createMail(model.Mail{
			Character: char,
			MailID:    int32(10 + i),
		})
	}
	// when
	ids, err := model.FetchMailIDs(char.ID)
	assert.NoError(t, err)
	got := set.NewFromSlice(ids)
	want := set.NewFromSlice([]int32{10, 11, 12})
	assert.Equal(t, want, got)
}

func TestCanFetchMailsForCharacterAndLabel(t *testing.T) {
	// given
	model.TruncateTables()
	c := createCharacter()
	l1 := createMailLabel(model.MailLabel{Character: c})
	l2 := createMailLabel(model.MailLabel{Character: c})
	m1 := createMail(model.Mail{Character: c, Labels: []model.MailLabel{l1}, Timestamp: time.Now().Add(time.Second * -120)})
	m2 := createMail(model.Mail{Character: c, Labels: []model.MailLabel{l1}, Timestamp: time.Now().Add(time.Second * -60)})
	createMail(model.Mail{Character: c, Labels: []model.MailLabel{l2}})
	// when
	mm, err := model.FetchMailsForLabel(c.ID, l1.LabelID)
	// then
	if assert.NoError(t, err) {
		var gotIDs []int32
		for _, m := range mm {
			gotIDs = append(gotIDs, m.MailID)
		}
		wantIDs := []int32{m2.MailID, m1.MailID}
		assert.Equal(t, wantIDs, gotIDs)
	}
}

func TestCanFetchAllMailsForCharacter(t *testing.T) {
	// given
	model.TruncateTables()
	c := createCharacter()
	l1 := createMailLabel(model.MailLabel{Character: c})
	l2 := createMailLabel(model.MailLabel{Character: c})
	m1 := createMail(model.Mail{Character: c, Labels: []model.MailLabel{l1}, Timestamp: time.Now().Add(time.Second * -120)})
	m2 := createMail(model.Mail{Character: c, Labels: []model.MailLabel{l1}, Timestamp: time.Now().Add(time.Second * -60)})
	m3 := createMail(model.Mail{Character: c, Labels: []model.MailLabel{l2}, Timestamp: time.Now().Add(time.Second * -240)})
	// when
	mm, err := model.FetchMailsForLabel(c.ID, model.AllMailsLabelID)
	// then
	if assert.NoError(t, err) {
		var gotIDs []int32
		for _, m := range mm {
			gotIDs = append(gotIDs, m.MailID)
		}
		wantIDs := []int32{m2.MailID, m1.MailID, m3.MailID}
		assert.Equal(t, wantIDs, gotIDs)
	}
}
func TestFetchMailsForLabelReturnEmptyWhenNoMatch(t *testing.T) {
	// given
	model.TruncateTables()
	c := createCharacter()
	// when
	mm, err := model.FetchMailsForLabel(c.ID, 99)
	// then
	if assert.NoError(t, err) {
		assert.Empty(t, mm)
	}
}

func TestCanCreateMailWithLabelsForOtherCharacter(t *testing.T) {
	// given
	model.TruncateTables()
	c1 := createCharacter()
	l1 := createMailLabel(model.MailLabel{Character: c1, LabelID: 1})
	createMail(model.Mail{Character: c1, Labels: []model.MailLabel{l1}})
	c2 := createCharacter()
	l2 := createMailLabel(model.MailLabel{Character: c2, LabelID: 1})
	// when
	from := createEveEntity()
	m := model.Mail{
		Body:      "body",
		From:      from,
		MailID:    7,
		Character: c2,
		Subject:   "subject",
		Labels:    []model.MailLabel{l2},
		Timestamp: time.Now(),
	}
	assert.NoError(t, m.Create())
	// when
	mm, err := model.FetchMailsForLabel(c2.ID, l2.LabelID)
	if assert.NoError(t, err) {
		assert.Len(t, mm, 1)
	}
}
