package esi_test

import (
	"example/esiapp/internal/api/esi"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestCanFetchMailLabels(t *testing.T) {
	// given
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	json := `
	{
		"labels": [
			{
			"color": "#660066",
			"label_id": 16,
			"name": "PINK",
			"unread_count": 4
			},
			{
			"color": "#ffffff",
			"label_id": 17,
			"name": "WHITE",
			"unread_count": 1
			}
		],
		"total_unread_count": 5
	}`

	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/latest/characters/1/mail/labels/",
		httpmock.NewStringResponder(200, json),
	)

	c := &http.Client{}

	// when
	r, err := esi.FetchMailLabels(c, 1, "token")

	// then
	assert.Nil(t, err)
	assert.Equal(t, 1, httpmock.GetTotalCallCount())

	assert.Equal(t, int32(5), r.TotalUnreadCount)
	assert.Len(t, r.Labels, 2)
	assert.Equal(t, r.Labels[0], esi.MailLabel{LabelID: 16, Name: "PINK", Color: "#660066", UnreadCount: 4})
}
