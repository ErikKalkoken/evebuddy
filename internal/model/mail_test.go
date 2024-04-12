package model_test

import (
	"database/sql"
	"example/evebuddy/internal/factory"

	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/model"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMailCreate(t *testing.T) {
	t.Run("can create new", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := factory.CreateCharacter()
		f := factory.CreateEveEntity()
		r := factory.CreateEveEntity()
		l := factory.CreateMailLabel(model.MailLabel{Character: c})
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
		m2, err := model.FetchMail(c.ID, m.MailID)
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
	})
	t.Run("should return error when no character ID", func(t *testing.T) {
		// given
		model.TruncateTables()
		from := factory.CreateEveEntity()
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
	})
	t.Run("should return error when no from ID", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := factory.CreateCharacter()
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
	})
}

func TestFetchMailIDs(t *testing.T) {
	// given
	model.TruncateTables()
	char := factory.CreateCharacter()
	for i := range 3 {
		factory.CreateMail(model.Mail{
			Character: char,
			MailID:    int32(10 + i),
		})
	}
	// when
	ids, err := model.FetchMailIDs(char.ID)
	// then
	assert.NoError(t, err)
	got := set.NewFromSlice(ids)
	want := set.NewFromSlice([]int32{10, 11, 12})
	assert.Equal(t, want, got)
}

func TestFetchMailsForLabel(t *testing.T) {
	t.Run("should return mail for selected label only", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := factory.CreateCharacter()
		l1 := factory.CreateMailLabel(model.MailLabel{Character: c})
		l2 := factory.CreateMailLabel(model.MailLabel{Character: c})
		m1 := factory.CreateMail(model.Mail{Character: c, Labels: []model.MailLabel{l1}, Timestamp: time.Now().Add(time.Second * -120)})
		m2 := factory.CreateMail(model.Mail{Character: c, Labels: []model.MailLabel{l1}, Timestamp: time.Now().Add(time.Second * -60)})
		factory.CreateMail(model.Mail{Character: c, Labels: []model.MailLabel{l2}})
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
	})
	t.Run("can fetch for all labels", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := factory.CreateCharacter()
		l1 := factory.CreateMailLabel(model.MailLabel{Character: c})
		l2 := factory.CreateMailLabel(model.MailLabel{Character: c})
		m1 := factory.CreateMail(model.Mail{Character: c, Labels: []model.MailLabel{l1}, Timestamp: time.Now().Add(time.Second * -120)})
		m2 := factory.CreateMail(model.Mail{Character: c, Labels: []model.MailLabel{l1}, Timestamp: time.Now().Add(time.Second * -60)})
		m3 := factory.CreateMail(model.Mail{Character: c, Labels: []model.MailLabel{l2}, Timestamp: time.Now().Add(time.Second * -240)})
		m4 := factory.CreateMail(model.Mail{Character: c, Timestamp: time.Now().Add(time.Second * -360)})
		// when
		mm, err := model.FetchMailsForLabel(c.ID, model.LabelAll)
		// then
		if assert.NoError(t, err) {
			var gotIDs []int32
			for _, m := range mm {
				gotIDs = append(gotIDs, m.MailID)
			}
			wantIDs := []int32{m2.MailID, m1.MailID, m3.MailID, m4.MailID}
			assert.Equal(t, wantIDs, gotIDs)
		}
	})
	t.Run("should return mail without label", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := factory.CreateCharacter()
		l := factory.CreateMailLabel(model.MailLabel{Character: c})
		factory.CreateMail(model.Mail{Character: c, Labels: []model.MailLabel{l}, Timestamp: time.Now().Add(time.Second * -120)})
		m := factory.CreateMail(model.Mail{Character: c})
		// when
		mm, err := model.FetchMailsForLabel(c.ID, model.LabelNone)
		// then
		if assert.NoError(t, err) {
			var gotIDs []int32
			for _, m := range mm {
				gotIDs = append(gotIDs, m.MailID)
			}
			wantIDs := []int32{m.MailID}
			assert.Equal(t, wantIDs, gotIDs)
		}
	})
	t.Run("should return empty when no match", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := factory.CreateCharacter()
		// when
		mm, err := model.FetchMailsForLabel(c.ID, 99)
		// then
		if assert.NoError(t, err) {
			assert.Empty(t, mm)
		}
	})
	t.Run("different characters can have same label ID", func(t *testing.T) {
		// given
		model.TruncateTables()
		c1 := factory.CreateCharacter()
		l1 := factory.CreateMailLabel(model.MailLabel{Character: c1, LabelID: 1})
		factory.CreateMail(model.Mail{Character: c1, Labels: []model.MailLabel{l1}})
		c2 := factory.CreateCharacter()
		l2 := factory.CreateMailLabel(model.MailLabel{Character: c2, LabelID: 1})
		// when
		from := factory.CreateEveEntity()
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
	})
}

func TestFetchMailsFoList(t *testing.T) {
	t.Run("should return mail for selected list only", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := factory.CreateCharacter()
		l1 := factory.CreateMailList(model.MailList{Character: c})
		m1 := factory.CreateMail(model.Mail{Character: c, Recipients: []model.EveEntity{l1.EveEntity}})
		l2 := factory.CreateMailList(model.MailList{Character: c})
		factory.CreateMail(model.Mail{Character: c, Recipients: []model.EveEntity{l2.EveEntity}})
		factory.CreateMail(model.Mail{Character: c})
		// when
		mm, err := model.FetchMailsForList(c.ID, l1.EveEntityID)
		// then
		if assert.NoError(t, err) {
			var gotIDs []int32
			for _, m := range mm {
				gotIDs = append(gotIDs, m.MailID)
			}
			wantIDs := []int32{m1.MailID}
			assert.Equal(t, wantIDs, gotIDs)
		}
	})
}

func TestDeleteMail(t *testing.T) {
	t.Run("can delete existing mail", func(t *testing.T) {
		// given
		model.TruncateTables()
		m := factory.CreateMail()
		// when
		c, err := model.DeleteMail(m.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 1, c)
		}
	})
}

func TestFetchMailLabelUnreadCounts(t *testing.T) {
	// given
	model.TruncateTables()
	c := factory.CreateCharacter()
	corp := factory.CreateMailLabel(model.MailLabel{Character: c, LabelID: model.LabelCorp})
	inbox := factory.CreateMailLabel(model.MailLabel{Character: c, LabelID: model.LabelInbox})
	factory.CreateMailLabel(model.MailLabel{Character: c, LabelID: model.LabelAlliance})
	factory.CreateMail(model.Mail{Character: c, Labels: []model.MailLabel{inbox}, IsRead: false})
	factory.CreateMail(model.Mail{Character: c, Labels: []model.MailLabel{corp}, IsRead: true})
	factory.CreateMail(model.Mail{Character: c, Labels: []model.MailLabel{corp}, IsRead: false})
	factory.CreateMail(model.Mail{Character: c, Labels: []model.MailLabel{corp}, IsRead: false})
	factory.CreateMail(model.Mail{Character: c})
	// when
	r, err := model.FetchMailLabelUnreadCounts(c.ID)
	if assert.NoError(t, err) {
		assert.Equal(t, map[int32]int{model.LabelCorp: 2, model.LabelInbox: 1}, r)
	}
}

func TestFetchMailListUnreadCounts(t *testing.T) {
	// given
	model.TruncateTables()
	c := factory.CreateCharacter()
	l1 := factory.CreateMailList(model.MailList{Character: c})
	factory.CreateMailList(model.MailList{Character: c})
	factory.CreateMail(model.Mail{Character: c, Recipients: []model.EveEntity{l1.EveEntity}, IsRead: false})
	factory.CreateMail(model.Mail{Character: c, Recipients: []model.EveEntity{l1.EveEntity}, IsRead: true})
	factory.CreateMail(model.Mail{Character: c})
	// when
	r, err := model.FetchMailListUnreadCounts(c.ID)
	if assert.NoError(t, err) {
		assert.Equal(t, map[int32]int{l1.EveEntityID: 1}, r)
	}
}

func TestMailSave(t *testing.T) {
	t.Run("can save updates", func(t *testing.T) {
		// given
		model.TruncateTables()
		m := factory.CreateMail(model.Mail{IsRead: false})
		m.IsRead = true
		// when
		err := m.Save()
		// then
		if assert.NoError(t, err) {
			m2, err := model.FetchMail(m.CharacterID, m.MailID)
			if assert.NoError(t, err) {
				assert.True(t, m2.IsRead)
			}
		}
	})
	t.Run("should return error when no ID", func(t *testing.T) {
		// given
		model.TruncateTables()
		m := model.Mail{}
		// when
		err := m.Save()
		// then
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}
