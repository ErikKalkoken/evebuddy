package characterservice_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
)

func TestUpdateMail(t *testing.T) {
	xesi.ActivateRateLimiterMock()
	defer xesi.DeactivateRateLimiterMock()
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	xesi.ActivateRateLimiterMock()
	defer xesi.DeactivateRateLimiterMock()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("Can fetch new mail", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c1 := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c1.ID})
		e1 := factory.CreateEveEntityCharacter()
		e2 := factory.CreateEveEntityCharacter()
		m1 := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList})
		factory.CreateCharacterMailList(c1.ID) // obsolete
		c2 := factory.CreateCharacterFull()
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
		mailID := 7
		labelIDs := []int32{16, 32}
		timestamp := "2015-09-30T16:07:00Z"
		subject := "test"
		dataHeader := []map[string]any{
			{
				"from":       e1.ID,
				"is_read":    true,
				"labels":     labelIDs,
				"mail_id":    mailID,
				"recipients": recipients,
				"subject":    subject,
				"timestamp":  timestamp,
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/", c1.ID),
			httpmock.NewJsonResponderOrPanic(200, dataHeader),
		)
		dataMail := map[string]any{
			"body":       "blah blah blah",
			"from":       e1.ID,
			"labels":     labelIDs,
			"read":       true,
			"recipients": recipients,
			"subject":    "test",
			"timestamp":  timestamp,
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/%d/", c1.ID, mailID),
			httpmock.NewJsonResponderOrPanic(200, dataMail),
		)
		dataMailList := []map[string]any{
			{
				"mailing_list_id": m1.ID,
				"name":            "test_mailing_list",
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/lists/", c1.ID),
			httpmock.NewJsonResponderOrPanic(200, dataMailList),
		)
		dataMailLabel := map[string]any{
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
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/mail/labels/", c1.ID),
			httpmock.NewJsonResponderOrPanic(200, dataMailLabel),
		)
		// when
		_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c1.ID,
			Section:     app.SectionCharacterMailLabels,
		})
		if assert.NoError(t, err) {
			_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
				CharacterID: c1.ID,
				Section:     app.SectionCharacterMailLists,
			})
			if assert.NoError(t, err) {
				_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
					CharacterID: c1.ID,
					Section:     app.SectionCharacterMails,
				})
				// then
				if assert.NoError(t, err) {
					m, err := s.GetMail(ctx, c1.ID, int32(mailID))
					if assert.NoError(t, err) {
						assert.Equal(t, "blah blah blah", m.Body)
					}
					labels, err := st.ListCharacterMailLabelsOrdered(ctx, c1.ID)
					if assert.NoError(t, err) {
						got := set.Of[int32]()
						for _, l := range labels {
							got.Add(l.LabelID)
						}
						want := set.Of(labelIDs...)
						assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
					}
					lists, err := st.ListCharacterMailListsOrdered(ctx, c2.ID)
					if assert.NoError(t, err) {
						got := set.Of[int32]()
						for _, l := range lists {
							got.Add(l.ID)
						}
						want := set.Of(m2.ID)
						assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
					}
				}
			}
		}
	})
	t.Run("Can update existing mail", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		e1 := factory.CreateEveEntityCharacter()
		e2 := factory.CreateEveEntityCharacter()
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: 16})
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: 32}) // obsolete
		m1 := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList})
		timestamp, _ := time.Parse("2006-01-02T15:04:05.999MST", "2015-09-30T16:07:00Z")
		mailID := int32(7)
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			Body:         "blah blah blah",
			CharacterID:  c.ID,
			FromID:       e1.ID,
			LabelIDs:     []int32{16},
			MailID:       mailID,
			IsRead:       false,
			RecipientIDs: []int32{e2.ID, m1.ID},
			Subject:      "test",
			Timestamp:    timestamp,
		})

		dataMailList := []map[string]any{
			{
				"mailing_list_id": m1.ID,
				"name":            "test_mailing_list",
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/lists/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, dataMailList),
		)
		dataMailLabel := map[string]any{
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
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/mail/labels/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, dataMailLabel),
		)
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
		dataHeader := []map[string]any{
			{
				"from":       e1.ID,
				"is_read":    true,
				"labels":     []int{32},
				"mail_id":    mailID,
				"recipients": recipients,
				"subject":    "test",
				"timestamp":  "2015-09-30T16:07:00Z",
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, dataHeader),
		)
		// when
		_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterMails,
		})
		// then
		if assert.NoError(t, err) {
			m, err := s.GetMail(ctx, c.ID, mailID)
			if assert.NoError(t, err) {
				assert.Equal(t, "blah blah blah", m.Body)
				assert.True(t, m.IsRead)
				assert.Len(t, m.Labels, 1)
				assert.Equal(t, int32(32), m.Labels[0].LabelID)
				assert.Len(t, m.Recipients, 2)
			}
			labels, err := st.ListCharacterMailLabelsOrdered(ctx, c.ID)
			if assert.NoError(t, err) {
				got := set.Of[int32]()
				for _, l := range labels {
					got.Add(l.LabelID)
				}
				want := set.Of[int32](16, 32)
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
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
			n := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
				IsProcessed: tc.isProcessed,
				Timestamp:   tc.timestamp,
			})
			var sendCount int
			// when
			err := cs.NotifyMails(ctx, n.CharacterID, earliest, func(title string, content string) {
				sendCount++
			})
			// then
			if assert.NoError(t, err) {
				assert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}

func TestSendMail(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	xesi.ActivateRateLimiterMock()
	defer xesi.DeactivateRateLimiterMock()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := characterservice.NewFake(st)
	t.Run("Can send mail", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		ctx = xesi.NewContextWithCharacterID(ctx, c.ID)
		r := factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/", c.ID),
			httpmock.NewJsonResponderOrPanic(201, 123))

		// when
		mailID, err := s.SendMail(ctx, c.ID, "subject", []*app.EveEntity{r}, "body")
		// then
		if assert.NoError(t, err) {
			m, err := s.GetMail(ctx, c.ID, mailID)
			if assert.NoError(t, err) {
				assert.Equal(t, "body", m.Body)
			}
		}
	})
}
