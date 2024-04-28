package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/antihax/goesi/esi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"example/evebuddy/internal/helper/testutil"
	"example/evebuddy/internal/model"
)

func TestCanFetchManyMailHeaders(t *testing.T) {
	// given
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	s := NewService(nil)
	var objs []esi.GetCharactersCharacterIdMail200Ok
	var mailIDs []int32
	for i := range 55 {
		id := int32(1000 - i)
		mailIDs = append(mailIDs, id)
		o := esi.GetCharactersCharacterIdMail200Ok{
			From:       90000001,
			IsRead:     true,
			Labels:     []int32{3},
			MailId:     id,
			Recipients: []esi.GetCharactersCharacterIdMailRecipient{{RecipientId: 90000002, RecipientType: "character"}},
			Subject:    fmt.Sprintf("Test Mail %d", id),
			Timestamp:  time.Now(),
		}
		objs = append(objs, o)
	}
	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/v1/characters/1/mail/",
		func(r *http.Request) (*http.Response, error) {
			resp, err := httpmock.NewJsonResponse(200, objs[:50])
			return resp, err
		},
	)
	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/v1/characters/1/mail/?last_mail_id=951",
		func(r *http.Request) (*http.Response, error) {
			resp, err := httpmock.NewJsonResponse(200, objs[50:])
			return resp, err
		},
	)
	token := model.Token{AccessToken: "abc", CharacterID: 1, ExpiresAt: time.Now().Add(time.Minute * 10)}

	// when
	mails, err := s.listMailHeaders(ctx, &token)

	// then
	if assert.NoError(t, err) {
		assert.Equal(t, 2, httpmock.GetTotalCallCount())
		assert.Len(t, mails, 55)

		newIDs := make([]int32, 0, 55)
		for _, m := range mails {
			newIDs = append(newIDs, m.MailId)
		}
		assert.Equal(t, mailIDs, newIDs)
	}
}

func TestUpdateMailLabel(t *testing.T) {
	// given
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	s := NewService(r)
	t.Run("should create new mail labels", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateMyCharacter()
		token := factory.CreateToken(model.Token{CharacterID: c.ID})
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
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, dataMailLabel)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			})
		// when
		err := s.updateMailLabels(ctx, token)
		// then
		if assert.NoError(t, err) {
			labels, err := r.ListMailLabelsOrdered(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, labels, 2)
			}
		}
	})
	t.Run("should update existing mail labels", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateMyCharacter()
		token := factory.CreateToken(model.Token{CharacterID: c.ID})
		l1 := factory.CreateMailLabel(model.MailLabel{
			MyCharacterID: c.ID,
			LabelID:       16,
			Name:          "BLACK",
			Color:         "#000000",
			UnreadCount:   99,
		})
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
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, dataMailLabel)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			})
		// when
		err := s.updateMailLabels(ctx, token)
		// then
		if assert.NoError(t, err) {
			l2, err := r.GetMailLabel(ctx, c.ID, l1.LabelID)
			if assert.NoError(t, err) {
				assert.Equal(t, "PINK", l2.Name)
				assert.Equal(t, "#660066", l2.Color)
				assert.Equal(t, 4, l2.UnreadCount)
			}
		}
	})
}
