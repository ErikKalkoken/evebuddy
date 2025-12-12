package characterservice_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMail(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("Can fetch new mail", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c1 := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c1.ID})
		e1 := factory.CreateEveEntityCharacter()
		e2 := factory.CreateEveEntityCharacter()
		m1 := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList})
		factory.CreateCharacterMailList(c1.ID) // obsolete
		c2 := factory.CreateCharacter()
		m2 := factory.CreateCharacterMailList(c2.ID) // not obsolete
		recipients := []map[string]any{
			{
				"recipient_id":   e2.ID,
				"recipient_type": "character",
			},
			{
				"recipient_id":   m1.ID,
				"recipient_type": "mail_list",
			},
		}
		mailID := int32(7)
		labelIDs := []int32{16, 32}
		timestamp := "2015-09-30T16:07:00Z"
		subject := "test"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/", c1.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"from":       e1.ID,
					"is_read":    true,
					"labels":     labelIDs,
					"mail_id":    mailID,
					"recipients": recipients,
					"subject":    subject,
					"timestamp":  timestamp,
				},
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/%d/", c1.ID, mailID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"body":       "blah blah blah",
				"from":       e1.ID,
				"labels":     labelIDs,
				"read":       true,
				"recipients": recipients,
				"subject":    "test",
				"timestamp":  timestamp,
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/lists/", c1.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"mailing_list_id": m1.ID,
					"name":            "test_mailing_list",
				},
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/mail/labels/", c1.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"labels": []map[string]any{
					{
						"color":        "#660066",
						"label_id":     16,
						"name":         "PINK",
						"unread_count": 4,
					},
					{
						"color":        "#FFFFFF",
						"label_id":     32,
						"name":         "WHITE",
						"unread_count": 0,
					},
				},
				"total_unread_count": 4,
			}),
		)
		// when
		_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c1.ID,
			Section:     app.SectionCharacterMailLabels,
		})
		require.NoError(t, err)
		_, err = s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c1.ID,
			Section:     app.SectionCharacterMailLists,
		})
		require.NoError(t, err)
		_, err = s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c1.ID,
			Section:     app.SectionCharacterMailHeaders,
		})
		// then
		require.NoError(t, err)
		m, err := s.GetMail(ctx, c1.ID, mailID)
		require.NoError(t, err)
		assert.Equal(t, subject, m.Subject)
		labels, err := st.ListCharacterMailLabelsOrdered(ctx, c1.ID)
		require.NoError(t, err)
		got := set.Of[int32]()
		for _, l := range labels {
			got.Add(l.LabelID)
		}
		want := set.Of(labelIDs...)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)

		lists, err := st.ListCharacterMailListsOrdered(ctx, c2.ID)
		require.NoError(t, err)
		got2 := set.Of[int32]()
		for _, l := range lists {
			got2.Add(l.ID)
		}
		want2 := set.Of(m2.ID)
		assert.True(t, got2.Equal(want2), "got %q, wanted %q", got2, want2)

	})
	t.Run("Can update existing mail", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		e1 := factory.CreateEveEntityCharacter()
		e2 := factory.CreateEveEntityCharacter()
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: 16})
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: 32}) // obsolete
		m1 := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList})
		timestamp, _ := time.Parse("2006-01-02T15:04:05.999MST", "2015-09-30T16:07:00Z")
		mailID := int32(7)
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			Body:         optional.New("blah blah blah"),
			CharacterID:  c.ID,
			FromID:       e1.ID,
			LabelIDs:     []int32{16},
			MailID:       mailID,
			IsRead:       false,
			RecipientIDs: []int32{e2.ID, m1.ID},
			Subject:      "test",
			Timestamp:    timestamp,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/lists/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"mailing_list_id": m1.ID,
					"name":            "test_mailing_list",
				},
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/mail/labels/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"labels": []map[string]any{
					{
						"color":        "#660066",
						"label_id":     16,
						"name":         "PINK",
						"unread_count": 4,
					},
					{
						"color":        "#FFFFFF",
						"label_id":     32,
						"name":         "WHITE",
						"unread_count": 0,
					},
				},
				"total_unread_count": 4,
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"from":    e1.ID,
					"is_read": true,
					"labels":  []int{32},
					"mail_id": mailID,
					"recipients": []map[string]any{
						{
							"recipient_id":   e2.ID,
							"recipient_type": "character",
						},
						{
							"recipient_id":   m1.ID,
							"recipient_type": "mail_list",
						},
					},
					"subject":   "test",
					"timestamp": "2015-09-30T16:07:00Z",
				},
			}),
		)
		// when
		_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterMailHeaders,
		})
		// then
		require.NoError(t, err)
		m, err := s.GetMail(ctx, c.ID, mailID)
		require.NoError(t, err)
		assert.Equal(t, "blah blah blah", m.Body.ValueOrZero())
		assert.True(t, m.IsRead)
		assert.Len(t, m.Labels, 1)
		assert.Equal(t, int32(32), m.Labels[0].LabelID)
		assert.Len(t, m.Recipients, 2)
		labels, err := st.ListCharacterMailLabelsOrdered(ctx, c.ID)
		require.NoError(t, err)
		got := set.Of[int32]()
		for _, l := range labels {
			got.Add(l.LabelID)
		}
		want := set.Of[int32](16, 32)
		assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
	})
}

func TestUpdateMailBodies(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := characterservice.NewFake(st)
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
			"body":       m.Body.MustValue(),
			"from":       m.From,
			"labels":     m.LabelIDs(),
			"read":       true,
			"recipients": recipients,
			"subject":    m.Subject,
			"timestamp":  m.Timestamp.Format(app.DateTimeFormatESI),
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
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/%d/", mail.CharacterID, mail.MailID),
			httpmock.NewJsonResponderOrPanic(200, makeMailData(mail)),
		)
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: mail.CharacterID})
		// when
		body, err := s.UpdateMailBodyESI(ctx, mail.CharacterID, mail.MailID)
		require.NoError(t, err)
		assert.Equal(t, "body", body)
		mail2, err := s.GetMail(ctx, mail.CharacterID, mail.MailID)
		require.NoError(t, err)
		assert.Equal(t, "body", mail2.Body.ValueOrZero())
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
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/%d/", c.ID, mail1a.MailID),
			httpmock.NewJsonResponderOrPanic(200, makeMailData(mail1a)),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/%d/", c.ID, mail2a.MailID),
			httpmock.NewJsonResponderOrPanic(200, makeMailData(mail2a)),
		)
		// when
		aborted, err := s.DownloadMissingMailBodies(ctx, c.ID)
		require.NoError(t, err)
		require.False(t, aborted)
		mail1b, err := s.GetMail(ctx, c.ID, mail1a.MailID)
		require.NoError(t, err)
		assert.Equal(t, "body1", mail1b.Body.ValueOrZero())
		mail2b, err := s.GetMail(ctx, c.ID, mail2a.MailID)
		require.NoError(t, err)
		assert.Equal(t, "body2", mail2b.Body.ValueOrZero())
	})
}

func TestNotifyMails(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	cs := characterservice.NewFake(st)
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
			assert.Equal(t, tc.shouldNotify, sendCount == 1)
		})
	}
}

func TestSendMail(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := characterservice.NewFake(st)
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
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/", c.ID),
			httpmock.NewJsonResponderOrPanic(201, 123))

		// when
		mailID, err := s.SendMail(ctx, c.ID, "subject", []*app.EveEntity{r}, "body")
		// then
		require.NoError(t, err)
		m, err := s.GetMail(ctx, c.ID, mailID)
		require.NoError(t, err)
		assert.Equal(t, "body", m.Body.ValueOrZero())
	})
}
