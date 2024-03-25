package esi_test

import (
	"encoding/json"
	"example/esiapp/internal/esi"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestCanFetchSingleMailHeader(t *testing.T) {
	// given
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	fixture := `
		[
			{
			"from": 90000001,
			"is_read": true,
			"labels": [
				3
			],
			"mail_id": 7,
			"recipients": [
				{
				"recipient_id": 90000002,
				"recipient_type": "character"
				}
			],
			"subject": "Title for EVE Mail",
			"timestamp": "2015-09-30T16:07:00Z"
			}
		]`

	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/latest/characters/1/mail/?token=token",
		httpmock.NewStringResponder(200, fixture),
	)

	c := http.Client{}

	// when
	mails, err := esi.FetchMailHeaders(c, 1, "token", 0)

	// then
	assert.Nil(t, err)
	assert.Equal(t, 1, httpmock.GetTotalCallCount())
	assert.Len(t, mails, 1)
	expected := esi.MailHeader{
		FromID:     90000001,
		IsRead:     true,
		Labels:     []int32{3},
		ID:         7,
		Recipients: []esi.MailRecipient{{ID: 90000002, Type: "character"}},
		Subject:    "Title for EVE Mail",
		Timestamp:  "2015-09-30T16:07:00Z",
	}
	assert.Equal(t, expected, mails[0])

}

func jsonMarshal(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)

}

func TestCanFetchManyMailHeaders(t *testing.T) {
	// given
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	objs := []esi.MailHeader{}
	var mailIDs []int32
	for i := range 55 {
		id := int32(1000 - i)
		mailIDs = append(mailIDs, id)
		o := esi.MailHeader{
			FromID:     90000001,
			IsRead:     true,
			Labels:     []int32{3},
			ID:         id,
			Recipients: []esi.MailRecipient{{ID: 90000002, Type: "character"}},
			Subject:    fmt.Sprintf("Test Mail %d", id),
			Timestamp:  "2015-09-30T16:07:00Z",
		}
		objs = append(objs, o)
	}
	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/latest/characters/1/mail/?token=token",
		httpmock.NewStringResponder(200, jsonMarshal(objs[:50])),
	)
	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/latest/characters/1/mail/?last_mail_id=951&token=token",
		httpmock.NewStringResponder(200, jsonMarshal(objs[50:])),
	)

	c := http.Client{}

	// when
	mails, err := esi.FetchMailHeaders(c, 1, "token", 0)

	// then
	assert.Nil(t, err)
	assert.Equal(t, 2, httpmock.GetTotalCallCount())
	assert.Len(t, mails, 55)

	newIDs := make([]int32, 0, 55)
	for _, m := range mails {
		newIDs = append(newIDs, m.ID)
	}
	assert.Equal(t, mailIDs, newIDs)
}
