package esi

import (
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestCanFetchMailHeader(t *testing.T) {
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

	httpmock.RegisterResponder("GET", "https://esi.evetech.net/latest/characters/1/mail/?token=token",
		httpmock.NewStringResponder(200, fixture))

	c := &http.Client{}

	// when
	mails, err := FetchMailHeaders(c, 1, "token")

	// then
	assert.Nil(t, err)
	assert.Equal(t, 1, httpmock.GetTotalCallCount())
	assert.Equal(t, 1, len(mails))
	expected := MailHeader{
		FromID:     90000001,
		IsRead:     true,
		Labels:     []int32{3},
		ID:         7,
		Recipients: []MailRecipient{{ID: 90000002, Type: "character"}},
		Subject:    "Title for EVE Mail",
		Timestamp:  "2015-09-30T16:07:00Z",
	}
	assert.Equal(t, expected, mails[0])

}
