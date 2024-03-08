package esi

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
)

const maxHeadersPerPage = 50 // maximum header objects returned per page

// A mail recipient returned from ESI.
type MailRecipient struct {
	ID   int32  `json:"recipient_id"`
	Type string `json:"recipient_type"`
}

// A mail header returned from ESI.
type MailHeader struct {
	FromID     int32           `json:"from"`
	IsRead     bool            `json:"is_read"`
	ID         int32           `json:"mail_id"`
	Labels     []int32         `json:"labels"`
	Recipients []MailRecipient `json:"recipients"`
	Subject    string          `json:"subject"`
	Timestamp  string          `json:"timestamp"`
}

// FetchMailHeaders fetches all mail headers for a character from ESI and returns them.
func FetchMailHeaders(httpClient http.Client, characterID int32, tokenString string) ([]MailHeader, error) {
	var result []MailHeader
	lastMailID := int32(0)
	for {
		objs, err := fetchMailHeadersPage(httpClient, characterID, tokenString, lastMailID)
		if err != nil {
			return nil, err
		}
		result = append(result, objs...)
		if len(objs) < maxHeadersPerPage {
			break
		}
		ids := make([]int32, 0)
		for _, o := range objs {
			ids = append(ids, o.ID)
		}
		lastMailID = slices.Min(ids)
	}

	return result, nil
}

func fetchMailHeadersPage(client http.Client, characterID int32, tokenString string, lastMailID int32) ([]MailHeader, error) {
	v := url.Values{}
	v.Set("token", tokenString)
	if lastMailID > 0 {
		v.Set("last_mail_id", strconv.Itoa(int(lastMailID)))
	}
	path := fmt.Sprintf("/characters/%d/mail/?%v", characterID, v.Encode())
	log.Printf("Fetching mail headers for character ID %d with last ID %d", characterID, lastMailID)
	resp, err := getESI(client, path)
	if err != nil {
		return nil, err
	}
	return unmarshalResponse[[]MailHeader](resp)
}
