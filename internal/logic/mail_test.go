package logic

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/antihax/goesi/esi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestCanFetchManyMailHeaders(t *testing.T) {
	// given
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

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
	token := Token{AccessToken: "abc", CharacterID: 1, ExpiresAt: time.Now().Add(time.Minute * 10)}

	// when
	mails, err := listMailHeaders(&token)

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
