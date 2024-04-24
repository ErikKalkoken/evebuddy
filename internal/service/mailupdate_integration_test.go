package service_test

import (
	"context"
	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/service"
	"example/evebuddy/internal/storage"
	"example/evebuddy/internal/testutil"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestUpdateMail(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := service.NewService(r)
	t.Run("Can fetch new mail", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateToken(model.Token{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(model.EveEntity{ID: 90000001})
		factory.CreateEveEntityCharacter(model.EveEntity{ID: 90000002})
		dataHeader := []map[string]any{
			{
				"from":    90000001,
				"is_read": true,
				"labels":  []int{16},
				"mail_id": 7,
				"recipients": []map[string]any{
					{
						"recipient_id":   90000002,
						"recipient_type": "character",
					},
				},
				"subject":   "test",
				"timestamp": "2015-09-30T16:07:00Z",
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/", c.ID),
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, dataHeader)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			})

		dataMail := map[string]any{
			"body":      "blah blah blah",
			"from":      90000001,
			"labels":    []int{16},
			"read":      true,
			"subject":   "test",
			"timestamp": "2015-09-30T16:07:00Z",
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/7/", c.ID),
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, dataMail)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			})

		dataMailList := []map[string]any{
			{
				"mailing_list_id": 1,
				"name":            "test_mailing_list",
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/lists/", c.ID),
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, dataMailList)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
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
		_, err := s.UpdateMail(c.ID)
		// then
		if assert.NoError(t, err) {
			m, err := s.GetMail(c.ID, 7)
			if assert.NoError(t, err) {
				assert.Equal(t, "blah blah blah", m.Body)
			}
		}
	})
	t.Run("Can update existing mail", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateToken(model.Token{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(model.EveEntity{ID: 90000001})
		factory.CreateEveEntityCharacter(model.EveEntity{ID: 90000002})
		factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID, LabelID: 16})
		factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID, LabelID: 32})
		timestamp, _ := time.Parse("2006-01-02T15:04:05.999MST", "2015-09-30T16:07:00Z")
		factory.CreateMail(storage.CreateMailParams{
			Body:        "blah blah blah",
			CharacterID: c.ID,
			FromID:      90000001,
			LabelIDs:    []int32{16},
			MailID:      7,
			IsRead:      false,
			Subject:     "test",
			Timestamp:   timestamp,
		})
		dataHeader := []map[string]any{
			{
				"from":    90000001,
				"is_read": true,
				"labels":  []int{32},
				"mail_id": 7,
				"recipients": []map[string]any{
					{
						"recipient_id":   90000002,
						"recipient_type": "character",
					},
				},
				"subject":   "test",
				"timestamp": "2015-09-30T16:07:00Z",
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/", c.ID),
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, dataHeader)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			})

		dataMail := map[string]any{
			"body":      "blah blah blah",
			"from":      90000001,
			"labels":    []int{32},
			"read":      true,
			"subject":   "test",
			"timestamp": "2015-09-30T16:07:00Z",
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/7/", c.ID),
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, dataMail)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			})

		dataMailList := []map[string]any{
			{
				"mailing_list_id": 1,
				"name":            "test_mailing_list",
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/lists/", c.ID),
			func(req *http.Request) (*http.Response, error) {
				resp, err := httpmock.NewJsonResponse(200, dataMailList)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
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
		_, err := s.UpdateMail(c.ID)
		// then
		if assert.NoError(t, err) {
			m, err := s.GetMail(c.ID, 7)
			if assert.NoError(t, err) {
				assert.Equal(t, "blah blah blah", m.Body)
				assert.True(t, m.IsRead)
				assert.Len(t, m.Labels, 1)
				assert.Equal(t, int32(32), m.Labels[0].LabelID)
			}
			labels, err := r.ListMailLabelsOrdered(context.Background(), c.ID)
			if assert.NoError(t, err) {
				got := set.New[int32]()
				for _, l := range labels {
					got.Add(l.LabelID)
				}
				want := set.NewFromSlice([]int32{32})
				assert.Equal(t, want, got)
			}
		}
	})
}
