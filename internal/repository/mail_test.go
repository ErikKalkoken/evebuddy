package repository_test

import (
	"context"
	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/repository"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMailCreate(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		f := factory.CreateEveEntity()
		recipient := factory.CreateEveEntity()
		label := factory.CreateMailLabel(repository.MailLabel{CharacterID: c.ID})
		// when
		arg := repository.CreateMailParams{
			Body:         "body",
			CharacterID:  c.ID,
			FromID:       f.ID,
			IsRead:       false,
			LabelIDs:     []int32{label.LabelID},
			MailID:       42,
			RecipientIDs: []int32{recipient.ID},
			Subject:      "subject",
			Timestamp:    time.Now(),
		}
		_, err := r.CreateMail(ctx, arg)
		// then
		if assert.NoError(t, err) {
			m, err := r.GetMail(ctx, c.ID, 42)
			assert.NoError(t, err)
			assert.Equal(t, int32(42), m.MailID)
			assert.Equal(t, "body", m.Body)
			assert.Equal(t, f, m.From)
			assert.Equal(t, c.ID, m.CharacterID)
			assert.Equal(t, "subject", m.Subject)
			assert.False(t, m.Timestamp.IsZero())
			assert.Equal(t, []repository.EveEntity{recipient}, m.Recipients)
			assert.Equal(t, label.Name, m.Labels[0].Name)
			assert.Equal(t, label.LabelID, m.Labels[0].LabelID)
		}
	})
}

func TestMail(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		o := repository.Token{
			AccessToken:  "access",
			CharacterID:  int32(c.ID),
			ExpiresAt:    time.Now(),
			RefreshToken: "refresh",
			TokenType:    "xxx",
		}
		// when
		err := r.UpdateOrCreateToken(ctx, &o)
		// then
		assert.NoError(t, err)
		r, err := r.GetToken(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, o.AccessToken, r.AccessToken)
			assert.Equal(t, c.ID, r.CharacterID)
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		o := repository.Token{
			AccessToken:  "access",
			CharacterID:  int32(c.ID),
			ExpiresAt:    time.Now(),
			RefreshToken: "refresh",
			TokenType:    "xxx",
		}
		if err := r.UpdateOrCreateToken(ctx, &o); err != nil {
			panic(err)
		}
		o.AccessToken = "changed"
		// when
		err := r.UpdateOrCreateToken(ctx, &o)
		// then
		assert.NoError(t, err)
		r, err := r.GetToken(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, o.AccessToken, r.AccessToken)
			assert.Equal(t, c.ID, r.CharacterID)
		}
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		_, err := r.GetMail(ctx, c.ID, 99)
		// then
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
	t.Run("can list mail IDs", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		for i := range 3 {
			factory.CreateMail(repository.CreateMailParams{
				CharacterID: c.ID,
				MailID:      int32(10 + i),
			})
		}
		// when
		ids, err := r.ListMailIDs(ctx, c.ID)
		// then
		assert.NoError(t, err)
		got := set.NewFromSlice(ids)
		want := set.NewFromSlice([]int32{10, 11, 12})
		assert.Equal(t, want, got)
	})
}

// func TestFetchMailsForLabel(t *testing.T) {
// 	t.Run("should return mail for selected label only", func(t *testing.T) {
// 		// given
// 		repository.TruncateTables(db)
// 		c := factory.CreateCharacter()
// 		l1 := factory.CreateMailLabel(repository.MailLabel{Character: c})
// 		l2 := factory.CreateMailLabel(repository.MailLabel{Character: c})
// 		m1 := factory.CreateMail(repository.Mail{Character: c, Labels: []repository.MailLabel{l1}, Timestamp: time.Now().Add(time.Second * -120)})
// 		m2 := factory.CreateMail(repository.Mail{Character: c, Labels: []repository.MailLabel{l1}, Timestamp: time.Now().Add(time.Second * -60)})
// 		factory.CreateMail(repository.Mail{Character: c, Labels: []repository.MailLabel{l2}})
// 		// when
// 		mm, err := repository.ListMailsForLabel(c.ID, l1.LabelID)
// 		// then
// 		if assert.NoError(t, err) {
// 			var gotIDs []int32
// 			for _, m := range mm {
// 				gotIDs = append(gotIDs, m.MailID)
// 			}
// 			wantIDs := []int32{m2.MailID, m1.MailID}
// 			assert.Equal(t, wantIDs, gotIDs)
// 		}
// 	})
// 	t.Run("can fetch for all labels", func(t *testing.T) {
// 		// given
// 		repository.TruncateTables(db)
// 		c := factory.CreateCharacter()
// 		l1 := factory.CreateMailLabel(repository.MailLabel{Character: c})
// 		l2 := factory.CreateMailLabel(repository.MailLabel{Character: c})
// 		m1 := factory.CreateMail(repository.Mail{Character: c, Labels: []repository.MailLabel{l1}, Timestamp: time.Now().Add(time.Second * -120)})
// 		m2 := factory.CreateMail(repository.Mail{Character: c, Labels: []repository.MailLabel{l1}, Timestamp: time.Now().Add(time.Second * -60)})
// 		m3 := factory.CreateMail(repository.Mail{Character: c, Labels: []repository.MailLabel{l2}, Timestamp: time.Now().Add(time.Second * -240)})
// 		m4 := factory.CreateMail(repository.Mail{Character: c, Timestamp: time.Now().Add(time.Second * -360)})
// 		// when
// 		mm, err := repository.ListMailsForLabel(c.ID, repository.LabelAll)
// 		// then
// 		if assert.NoError(t, err) {
// 			var gotIDs []int32
// 			for _, m := range mm {
// 				gotIDs = append(gotIDs, m.MailID)
// 			}
// 			wantIDs := []int32{m2.MailID, m1.MailID, m3.MailID, m4.MailID}
// 			assert.Equal(t, wantIDs, gotIDs)
// 		}
// 	})
// 	t.Run("should return mail without label", func(t *testing.T) {
// 		// given
// 		repository.TruncateTables(db)
// 		c := factory.CreateCharacter()
// 		l := factory.CreateMailLabel(repository.MailLabel{Character: c})
// 		factory.CreateMail(repository.Mail{Character: c, Labels: []repository.MailLabel{l}, Timestamp: time.Now().Add(time.Second * -120)})
// 		m := factory.CreateMail(repository.Mail{Character: c})
// 		// when
// 		mm, err := repository.ListMailsForLabel(c.ID, repository.LabelNone)
// 		// then
// 		if assert.NoError(t, err) {
// 			var gotIDs []int32
// 			for _, m := range mm {
// 				gotIDs = append(gotIDs, m.MailID)
// 			}
// 			wantIDs := []int32{m.MailID}
// 			assert.Equal(t, wantIDs, gotIDs)
// 		}
// 	})
// 	t.Run("should return empty when no match", func(t *testing.T) {
// 		// given
// 		repository.TruncateTables(db)
// 		c := factory.CreateCharacter()
// 		// when
// 		mm, err := repository.ListMailsForLabel(c.ID, 99)
// 		// then
// 		if assert.NoError(t, err) {
// 			assert.Empty(t, mm)
// 		}
// 	})
// 	t.Run("different characters can have same label ID", func(t *testing.T) {
// 		// given
// 		repository.TruncateTables(db)
// 		c1 := factory.CreateCharacter()
// 		l1 := factory.CreateMailLabel(repository.MailLabel{Character: c1, LabelID: 1})
// 		factory.CreateMail(repository.Mail{Character: c1, Labels: []repository.MailLabel{l1}})
// 		c2 := factory.CreateCharacter()
// 		l2 := factory.CreateMailLabel(repository.MailLabel{Character: c2, LabelID: 1})
// 		// when
// 		from := factory.CreateEveEntity()
// 		m := repository.Mail{
// 			Body:      "body",
// 			From:      from,
// 			MailID:    7,
// 			Character: c2,
// 			Subject:   "subject",
// 			Labels:    []repository.MailLabel{l2},
// 			Timestamp: time.Now(),
// 		}
// 		assert.NoError(t, m.Create())
// 		// when
// 		mm, err := repository.ListMailsForLabel(c2.ID, l2.LabelID)
// 		if assert.NoError(t, err) {
// 			assert.Len(t, mm, 1)
// 		}
// 	})
// }

// func TestFetchMailsFoList(t *testing.T) {
// 	t.Run("should return mail for selected list only", func(t *testing.T) {
// 		// given
// 		repository.TruncateTables(db)
// 		c := factory.CreateCharacter()
// 		l1 := factory.CreateMailList(repository.MailList{Character: c})
// 		m1 := factory.CreateMail(repository.Mail{Character: c, Recipients: []repository.EveEntity{l1.EveEntity}})
// 		l2 := factory.CreateMailList(repository.MailList{Character: c})
// 		factory.CreateMail(repository.Mail{Character: c, Recipients: []repository.EveEntity{l2.EveEntity}})
// 		factory.CreateMail(repository.Mail{Character: c})
// 		// when
// 		mm, err := repository.ListMailsForList(c.ID, l1.EveEntityID)
// 		// then
// 		if assert.NoError(t, err) {
// 			var gotIDs []int32
// 			for _, m := range mm {
// 				gotIDs = append(gotIDs, m.MailID)
// 			}
// 			wantIDs := []int32{m1.MailID}
// 			assert.Equal(t, wantIDs, gotIDs)
// 		}
// 	})
// }

// func TestDeleteMail(t *testing.T) {
// 	t.Run("can delete existing mail", func(t *testing.T) {
// 		// given
// 		repository.TruncateTables(db)
// 		m := factory.CreateMail()
// 		// when
// 		c, err := repository.DeleteMail(m.ID)
// 		// then
// 		if assert.NoError(t, err) {
// 			assert.Equal(t, 1, c)
// 		}
// 	})
// }

// func TestFetchMailLabelUnreadCounts(t *testing.T) {
// 	// given
// 	repository.TruncateTables(db)
// 	c := factory.CreateCharacter()
// 	corp := factory.CreateMailLabel(repository.MailLabel{Character: c, LabelID: repository.LabelCorp})
// 	inbox := factory.CreateMailLabel(repository.MailLabel{Character: c, LabelID: repository.LabelInbox})
// 	factory.CreateMailLabel(repository.MailLabel{Character: c, LabelID: repository.LabelAlliance})
// 	factory.CreateMail(repository.Mail{Character: c, Labels: []repository.MailLabel{inbox}, IsRead: false})
// 	factory.CreateMail(repository.Mail{Character: c, Labels: []repository.MailLabel{corp}, IsRead: true})
// 	factory.CreateMail(repository.Mail{Character: c, Labels: []repository.MailLabel{corp}, IsRead: false})
// 	factory.CreateMail(repository.Mail{Character: c, Labels: []repository.MailLabel{corp}, IsRead: false})
// 	factory.CreateMail(repository.Mail{Character: c})
// 	// when
// 	r, err := repository.GetMailLabelUnreadCounts(c.ID)
// 	if assert.NoError(t, err) {
// 		assert.Equal(t, map[int32]int{repository.LabelCorp: 2, repository.LabelInbox: 1}, r)
// 	}
// }

// func TestFetchMailListUnreadCounts(t *testing.T) {
// 	// given
// 	repository.TruncateTables(db)
// 	c := factory.CreateCharacter()
// 	l1 := factory.CreateMailList(repository.MailList{Character: c})
// 	factory.CreateMailList(repository.MailList{Character: c})
// 	factory.CreateMail(repository.Mail{Character: c, Recipients: []repository.EveEntity{l1.EveEntity}, IsRead: false})
// 	factory.CreateMail(repository.Mail{Character: c, Recipients: []repository.EveEntity{l1.EveEntity}, IsRead: true})
// 	factory.CreateMail(repository.Mail{Character: c})
// 	// when
// 	r, err := repository.GetMailListUnreadCounts(c.ID)
// 	if assert.NoError(t, err) {
// 		assert.Equal(t, map[int32]int{l1.EveEntityID: 1}, r)
// 	}
// }

// func TestMailSave(t *testing.T) {
// 	t.Run("can save updates", func(t *testing.T) {
// 		// given
// 		repository.TruncateTables(db)
// 		m := factory.CreateMail(repository.Mail{IsRead: false})
// 		m.IsRead = true
// 		// when
// 		err := m.Save()
// 		// then
// 		if assert.NoError(t, err) {
// 			m2, err := repository.GetMail(m.CharacterID, m.MailID)
// 			if assert.NoError(t, err) {
// 				assert.True(t, m2.IsRead)
// 			}
// 		}
// 	})
// 	t.Run("should return error when no ID", func(t *testing.T) {
// 		// given
// 		repository.TruncateTables(db)
// 		m := repository.Mail{}
// 		// when
// 		err := m.Save()
// 		// then
// 		assert.ErrorIs(t, err, sql.ErrNoRows)
// 	})
// }
