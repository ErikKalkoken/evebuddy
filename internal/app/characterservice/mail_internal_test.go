package characterservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fnt-eve/goesi-openapi/esi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateMail(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
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
		mailID := int64(7)
		labelIDs := []int64{16, 32}
		timestamp := "2015-09-30T16:07:00Z"
		subject := "test"
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail", c1.ID),
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
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/%d", c1.ID, mailID),
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
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/lists", c1.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"mailing_list_id": m1.ID,
					"name":            "test_mailing_list",
				},
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/labels", c1.ID),
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
		_, err := s.UpdateSectionIfNeeded(ctx, characterSectionUpdateParams{
			characterID: c1.ID,
			section:     app.SectionCharacterMailLabels,
		})
		require.NoError(t, err)
		_, err = s.UpdateSectionIfNeeded(ctx, characterSectionUpdateParams{
			characterID: c1.ID,
			section:     app.SectionCharacterMailLists,
		})
		require.NoError(t, err)
		_, err = s.UpdateSectionIfNeeded(ctx, characterSectionUpdateParams{
			characterID: c1.ID,
			section:     app.SectionCharacterMailHeaders,
		})
		// then
		require.NoError(t, err)
		m, err := s.GetMail(ctx, c1.ID, mailID)
		require.NoError(t, err)
		xassert.Equal(t, subject, m.Subject.ValueOrZero())
		labels, err := st.ListCharacterMailLabelsOrdered(ctx, c1.ID)
		require.NoError(t, err)
		got := set.Of[int64]()
		for _, l := range labels {
			got.Add(l.LabelID)
		}
		want := set.Of(labelIDs...)
		xassert.Equal(t, want, got)

		lists, err := st.ListCharacterMailListsOrdered(ctx, c2.ID)
		require.NoError(t, err)
		got2 := set.Of[int64]()
		for _, l := range lists {
			got2.Add(l.ID)
		}
		want2 := set.Of(m2.ID)
		xassert.Equal(t, want2, got2)
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
		mailID := int64(7)
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			Body:         optional.New("blah blah blah"),
			CharacterID:  c.ID,
			FromID:       e1.ID,
			LabelIDs:     []int64{16},
			MailID:       mailID,
			IsRead:       optional.New(false),
			RecipientIDs: []int64{e2.ID, m1.ID},
			Subject:      optional.New("test"),
			Timestamp:    timestamp,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/lists", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"mailing_list_id": m1.ID,
					"name":            "test_mailing_list",
				},
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/labels", c.ID),
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
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail", c.ID),
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
		_, err := s.UpdateSectionIfNeeded(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterMailHeaders,
		})
		// then
		require.NoError(t, err)
		m, err := s.GetMail(ctx, c.ID, mailID)
		require.NoError(t, err)
		xassert.Equal(t, "blah blah blah", m.Body.ValueOrZero())
		assert.True(t, m.IsRead.ValueOrZero())
		assert.Len(t, m.Labels, 1)
		xassert.Equal(t, int64(32), m.Labels[0].LabelID)
		assert.Len(t, m.Recipients, 2)
		labels, err := st.ListCharacterMailLabelsOrdered(ctx, c.ID)
		require.NoError(t, err)
		got := set.Of[int64]()
		for _, l := range labels {
			got.Add(l.LabelID)
		}
		want := set.Of[int64](16, 32)
		xassert.Equal(t, want, got)
	})
}

func TestCanFetchMailHeadersWithPaging(t *testing.T) {
	// given
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	var objs []esi.CharactersCharacterIdMailGetInner
	var mailIDs []int64
	for i := range 55 {
		id := int64(1000 - i)
		mailIDs = append(mailIDs, id)
		objs = append(objs, esi.CharactersCharacterIdMailGetInner{
			From:   new(int64(90000001)),
			IsRead: new(true),
			Labels: []int64{3},
			MailId: new(id),
			Recipients: []esi.PostCharactersCharacterIdMailRequestRecipientsInner{{
				RecipientId:   90000002,
				RecipientType: "character",
			}},
			Subject:   new(fmt.Sprintf("Test Mail %d", id)),
			Timestamp: new(time.Now()),
		})
	}
	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/characters/1/mail",
		httpmock.NewJsonResponderOrPanic(200, objs[:50]),
	)
	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/characters/1/mail?last_mail_id=951",
		httpmock.NewJsonResponderOrPanic(200, objs[50:]),
	)
	// when
	mails, err := s.fetchMailHeadersESI(ctx, 1, 1000)

	// then
	if assert.NoError(t, err) {
		xassert.Equal(t, 2, httpmock.GetTotalCallCount())
		assert.Len(t, mails, 55)

		newIDs := make([]int64, 0, 55)
		for _, m := range mails {
			newIDs = append(newIDs, m.MailID)
		}
		xassert.Equal(t, mailIDs, newIDs)
	}
}

func TestUpdateMailLabel(t *testing.T) {
	// given
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	s := NewFake(Params{Storage: st})
	t.Run("should create new mail labels", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/labels", c.ID),
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
		_, err := s.updateMailLabelsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterMailLabels,
		})
		// then
		if assert.NoError(t, err) {
			labels, err := st.ListCharacterMailLabelsOrdered(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, labels, 2)
			}
		}
	})
	t.Run("should update existing mail labels", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		l1 := factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: c.ID,
			LabelID:     16,
			Name:        optional.New("BLACK"),
			Color:       optional.New("#000000"),
			UnreadCount: optional.New[int64](99),
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/labels", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"labels": []map[string]any{{
					"color":        "#660066",
					"label_id":     16,
					"name":         "PINK",
					"unread_count": 4,
				}, {
					"color":        "#FFFFFF",
					"label_id":     32,
					"name":         "WHITE",
					"unread_count": 0,
				}},
				"total_unread_count": 4,
			}),
		)
		// when
		_, err := s.updateMailLabelsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterMailLabels,
		})
		// then
		if assert.NoError(t, err) {
			l2, err := st.GetCharacterMailLabel(ctx, c.ID, l1.LabelID)
			if assert.NoError(t, err) {
				xassert.Equal(t, "PINK", l2.Name.ValueOrZero())
				xassert.Equal(t, "#660066", l2.Color.ValueOrZero())
				xassert.Equal(t, 4, l2.UnreadCount.ValueOrZero())
			}
		}
	})
}
