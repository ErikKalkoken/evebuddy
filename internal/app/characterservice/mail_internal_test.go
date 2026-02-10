package characterservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fnt-eve/goesi-openapi/esi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCanFetchMailHeadersWithPaging(t *testing.T) {
	// given
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	var objs []esi.CharactersCharacterIdMailGetInner
	var mailIDs []int64
	for i := range 55 {
		id := int64(1000 - i)
		mailIDs = append(mailIDs, id)
		objs = append(objs, esi.CharactersCharacterIdMailGetInner{
			From:   testutil.Ptr(int64(90000001)),
			IsRead: testutil.Ptr(true),
			Labels: []int64{3},
			MailId: testutil.Ptr(id),
			Recipients: []esi.PostCharactersCharacterIdMailRequestRecipientsInner{{
				RecipientId:   90000002,
				RecipientType: "character",
			}},
			Subject:   testutil.Ptr(fmt.Sprintf("Test Mail %d", id)),
			Timestamp: testutil.Ptr(time.Now()),
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
	s := NewFake(st)
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
		_, err := s.updateMailLabelsESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterMailLabels,
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
		_, err := s.updateMailLabelsESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterMailLabels,
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
