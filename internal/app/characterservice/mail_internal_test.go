package characterservice

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/antihax/goesi/esi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestCanFetchMailHeadersWithPaging(t *testing.T) {
	// given
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
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
		httpmock.NewJsonResponderOrPanic(200, objs[:50]),
	)
	httpmock.RegisterResponder(
		"GET",
		"https://esi.evetech.net/v1/characters/1/mail/?last_mail_id=951",
		httpmock.NewJsonResponderOrPanic(200, objs[50:]),
	)
	// when
	mails, err := s.fetchMailHeadersESI(ctx, 1, 1000)

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
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/mail/labels/", c.ID),
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
			Name:        "BLACK",
			Color:       "#000000",
			UnreadCount: 99,
		})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/mail/labels/", c.ID),
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
			l2, err := st.GetCharacterMailLabel(ctx, c.ID, l1.LabelID)
			if assert.NoError(t, err) {
				assert.Equal(t, "PINK", l2.Name)
				assert.Equal(t, "#660066", l2.Color)
				assert.Equal(t, 4, l2.UnreadCount)
			}
		}
	})
}

func TestExtractRetryAfter(t *testing.T) {
	tests := []struct {
		name             string
		headers          http.Header
		expectError      bool
		expectedBase     time.Duration
		expectedGroup    string
		expectedErrorMsg string
	}{
		{
			name:             "Missing Header",
			headers:          http.Header{},
			expectError:      true,
			expectedBase:     0,
			expectedGroup:    "",
			expectedErrorMsg: "Retry-After header missing",
		},
		{
			name:             "Invalid Float Value",
			headers:          http.Header{"Retry-After": []string{"not-a-number"}},
			expectError:      true,
			expectedBase:     0,
			expectedGroup:    "",
			expectedErrorMsg: "invalid syntax",
		},
		{
			name:          "Valid Float with No Group",
			headers:       http.Header{"Retry-After": []string{"5.5"}},
			expectError:   false,
			expectedBase:  5500 * time.Millisecond,
			expectedGroup: "",
		},
		{
			name:          "Valid Integer with Group",
			headers:       http.Header{"Retry-After": []string{"10"}, "X-Ratelimit-Group": []string{"api-v2"}},
			expectError:   false,
			expectedBase:  10 * time.Second,
			expectedGroup: "api-v2",
		},
		{
			name:          "Valid Large Float",
			headers:       http.Header{"Retry-After": []string{"120.75"}},
			expectError:   false,
			expectedBase:  120750 * time.Millisecond,
			expectedGroup: "",
		},
	}
	const maxJitter = 999 * time.Millisecond
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			countdown, group, err := extractRetryAfter(tt.headers)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedGroup, group)
			minCountdown := tt.expectedBase
			maxCountdown := tt.expectedBase + maxJitter
			assert.True(t, countdown >= minCountdown, "Countdown %v is less than expected minimum %v", countdown, minCountdown)
			assert.True(t, countdown <= maxCountdown, "Countdown %v is greater than expected maximum %v", countdown, maxCountdown)
		})
	}
}
