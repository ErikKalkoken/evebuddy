package characterservice_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/fake"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateMailBodies(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := fake.NewCharacterService(characterservice.Params{Storage: st})
	ctx := context.Background()
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
		body, err := s.UpdateMailBodyESI(ctx, mail.CharacterID, mail.MailID)
		require.NoError(t, err)
		xassert.Equal(t, "body", body)
		mail2, err := s.GetMail(ctx, mail.CharacterID, mail.MailID)
		require.NoError(t, err)
		xassert.Equal(t, "body", mail2.Body.ValueOrZero())
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
		aborted, err := s.DownloadMissingMailBodies(ctx, c.ID)
		require.NoError(t, err)
		require.False(t, aborted)
		mail1b, err := s.GetMail(ctx, c.ID, mail1a.MailID)
		require.NoError(t, err)
		xassert.Equal(t, "body1", mail1b.Body.ValueOrZero())
		mail2b, err := s.GetMail(ctx, c.ID, mail2a.MailID)
		require.NoError(t, err)
		xassert.Equal(t, "body2", mail2b.Body.ValueOrZero())
	})
}

func TestNotifyMails(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	cs := fake.NewCharacterService(characterservice.Params{Storage: st})
	ctx := context.Background()
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
			err := cs.NotifyMails(ctx, n.CharacterID, earliest, func(title string, content string) {
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
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := fake.NewCharacterService(characterservice.Params{Storage: st})
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
		mailID, err := s.SendMail(ctx, c.ID, "subject", []*app.EveEntity{r}, "body")
		// then
		require.NoError(t, err)
		m, err := s.GetMail(ctx, c.ID, mailID)
		require.NoError(t, err)
		xassert.Equal(t, "body", m.Body.ValueOrZero())
	})
}
