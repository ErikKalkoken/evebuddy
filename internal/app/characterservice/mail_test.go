package characterservice_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func TestUpdateMailBodies(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})

	makeMailData := func(m *app.CharacterMail) map[string]any {
		recipients := make([]map[string]any, 0)
		for _, r := range m.Recipients {
			x := make(map[string]any)
			x["recipient_id"] = r.ID
			x["recipient_type"] = r.Category.String()
			recipients = append(recipients, x)
		}
		data := map[string]any{
			"labels":     m.LabelIDs(),
			"read":       true,
			"recipients": recipients,
			"timestamp":  m.Timestamp.Format(app.DateTimeFormatESI),
		}
		if x, ok := m.Body.Value(); ok {
			data["body"] = x
		}
		if x, ok := m.Subject.Value(); ok {
			data["subject"] = x
		}
		if m.From != nil {
			data["from"] = m.From.ID
		}
		return data
	}

	t.Run("Can update mail body", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		mail := factory.CreateCharacterMailWithBody()
		mail.Body.Set("body")
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/%d", mail.CharacterID, mail.MailID),
			httpmock.NewJsonResponderOrPanic(200, makeMailData(mail)),
		)
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: mail.CharacterID})

		// when
		body, err := s.UpdateMailBodyESI(t.Context(), mail.CharacterID, mail.MailID)

		// then
		require.NoError(t, err)
		xassert.Equal(t, "body", body)
		mail2, err := s.GetMail(t.Context(), mail.CharacterID, mail.MailID)
		require.NoError(t, err)
		xassert.EqualOptional(t, "body", mail2.Body)
	})

	t.Run("should delete local mail when it no longer exists on the server", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		mail := factory.CreateCharacterMail()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/%d", mail.CharacterID, mail.MailID),
			httpmock.NewJsonResponderOrPanic(http.StatusNotFound, map[string]any{"error": "some error"}),
		)
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: mail.CharacterID})

		// when
		_, err := s.UpdateMailBodyESI(t.Context(), mail.CharacterID, mail.MailID)

		// then
		assert.Error(t, err)
		_, err2 := s.GetMail(t.Context(), mail.CharacterID, mail.MailID)
		assert.ErrorIs(t, err2, app.ErrNotFound)

	})

	t.Run("Can download missing mail bodies", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		mail1a := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID})
		mail1a.Body.Set("body1")
		mail2a := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID})
		mail2a.Body.Set("body2")
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/%d", c.ID, mail1a.MailID),
			httpmock.NewJsonResponderOrPanic(200, makeMailData(mail1a)),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/%d", c.ID, mail2a.MailID),
			httpmock.NewJsonResponderOrPanic(200, makeMailData(mail2a)),
		)

		// when
		aborted, err := s.DownloadMissingMailBodies(t.Context(), c.ID)

		// then
		require.NoError(t, err)
		require.False(t, aborted)
		mail1b, err := s.GetMail(t.Context(), c.ID, mail1a.MailID)
		require.NoError(t, err)
		xassert.EqualOptional(t, "body1", mail1b.Body)
		mail2b, err := s.GetMail(t.Context(), c.ID, mail2a.MailID)
		require.NoError(t, err)
		xassert.EqualOptional(t, "body2", mail2b.Body)
	})
}

func TestNotifyMails(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	cs := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})

	now := time.Now().UTC()
	earliest := now.Add(-12 * time.Hour)
	cases := []struct {
		name         string
		timestamp    time.Time
		isProcessed  bool
		shouldNotify bool
	}{
		{"send unprocessed", now, false, true},
		{"don't send processed", now, true, false},
		{"don't send old", now.Add(-16 * time.Hour), false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			n := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
				IsProcessed: tc.isProcessed,
				Timestamp:   tc.timestamp,
			})
			var sendCount int
			// when
			err := cs.NotifyMails(t.Context(), n.CharacterID, earliest, func(title string, content string) {
				sendCount++
			})
			// then
			require.NoError(t, err)
			xassert.Equal(t, tc.shouldNotify, sendCount == 1)
		})
	}
}

func TestSendMail(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})

	t.Run("Can send mail", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		r := factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail", c.ID),
			httpmock.NewJsonResponderOrPanic(201, 123))

		// when
		mailID, err := s.SendMail(t.Context(), c.ID, "subject", []*app.EveEntity{r}, "body")

		// then
		require.NoError(t, err)
		m, err := s.GetMail(t.Context(), c.ID, mailID)
		require.NoError(t, err)
		xassert.EqualOptional(t, "body", m.Body)
	})

	t.Run("should handle 520 error", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		r := factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail", c.ID),
			httpmock.NewJsonResponderOrPanic(xgoesi.StatusTooManyErrors, map[string]any{
				"error": "ContactCostNotApproved",
				"details": map[string]any{
					"totalCost":       1,
					"totalCostISK":    []int{20, 1},
					"approvedCost":    0,
					"approvedCostISK": []int{20, 0},
				},
			}))

		// when
		_, err := s.SendMail(t.Context(), c.ID, "subject", []*app.EveEntity{r}, "body")

		// then
		require.Error(t, err)
	})
}

func TestDeleteMail(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})

	t.Run("can delete a mail", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"DELETE",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/%d", c.ID, m.MailID),
			httpmock.NewStringResponder(204, ""))
		// when
		err := s.DeleteMail(t.Context(), c.ID, m.MailID)
		// then
		require.NoError(t, err)
		_, err = s.GetMail(t.Context(), c.ID, m.MailID)
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestUpdateMailRead(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})

	t.Run("can mark a mail as read", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, IsRead: optional.New(false)})
		httpmock.RegisterResponder(
			"PUT",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/%d", c.ID, m.MailID),
			httpmock.NewStringResponder(204, ""))
		// when
		err := s.UpdateMailRead(t.Context(), c.ID, m.MailID, true)
		// then
		require.NoError(t, err)
		m2, err := s.GetMail(t.Context(), c.ID, m.MailID)
		require.NoError(t, err)
		xassert.EqualOptional(t, true, m2.IsRead)
	})
}

func TestGetAllMailUnreadCount(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	t.Run("can return unread mail count across all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, IsRead: optional.New(false)})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, IsRead: optional.New(true)})
		// when
		got, err := s.GetAllMailUnreadCount(t.Context())
		// then
		require.NoError(t, err)
		assert.Equal(t, 1, got)
	})
}

func TestGetMailCounts(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	t.Run("can return total and unread mail counts for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, IsRead: optional.New(false)})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, IsRead: optional.New(true)})
		// when
		total, unread, err := s.GetMailCounts(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Equal(t, 1, unread)
	})
}

func TestGetMailLabelUnreadCounts(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	t.Run("can return unread mail counts by label", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		label := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsRead:      optional.New(false),
			LabelIDs:    []int64{label.LabelID},
		})
		// when
		got, err := s.GetMailLabelUnreadCounts(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		assert.Equal(t, 1, got[label.LabelID])
	})
}

func TestGetMailListUnreadCounts(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	t.Run("can return unread mail counts by list", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		list := factory.CreateCharacterMailList(c.ID)
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			IsRead:       optional.New(false),
			RecipientIDs: []int64{list.ID},
		})
		// when
		got, err := s.GetMailListUnreadCounts(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		assert.Equal(t, 1, got[list.ID])
	})
}

func TestListMailLists(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	t.Run("can list mail lists for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		list := factory.CreateCharacterMailList(c.ID)
		// when
		got, err := s.ListMailLists(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		if assert.Len(t, got, 1) {
			assert.Equal(t, list.ID, got[0].ID)
		}
	})
}

func TestListMailLabelsOrdered(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	t.Run("can list mail labels for a character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		label := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID})
		// when
		got, err := s.ListMailLabelsOrdered(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		if assert.Len(t, got, 1) {
			assert.Equal(t, label.LabelID, got[0].LabelID)
		}
	})
}

func TestListMailHeadersForLabelOrdered(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	t.Run("can list mail headers for a label", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID})
		// when
		got, err := s.ListMailHeadersForLabelOrdered(t.Context(), c.ID, app.MailLabelAll)
		// then
		require.NoError(t, err)
		if assert.Len(t, got, 1) {
			assert.Equal(t, m.MailID, got[0].MailID)
		}
	})
}

func TestListMailHeadersForListOrdered(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	t.Run("can list mail headers for a mail list", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		list := factory.CreateCharacterMailList(c.ID)
		m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int64{list.ID},
		})
		// when
		got, err := s.ListMailHeadersForListOrdered(t.Context(), c.ID, list.ID)
		// then
		require.NoError(t, err)
		if assert.Len(t, got, 1) {
			assert.Equal(t, m.MailID, got[0].MailID)
		}
	})
}

func TestDownloadedBodiesPercentage(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	t.Run("can report total and missing mail body counts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID}) // no body
		// when
		total, missing, err := s.DownloadedBodiesPercentage(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, 1, missing)
	})
}
