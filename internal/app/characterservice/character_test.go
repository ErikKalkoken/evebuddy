package characterservice_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestGetCharacter(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("should return own error when object not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := cs.GetCharacter(ctx, 42)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return obj when found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x1 := factory.CreateCharacter()
		// when
		x2, err := cs.GetCharacter(ctx, x1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
}

func TestGetAnyCharacter(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("should return own error when object not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := cs.GetAnyCharacter(ctx)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return obj when found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x1 := factory.CreateCharacter()
		// when
		x2, err := cs.GetAnyCharacter(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
}

type ssoFake struct {
	token *app.Token
	err   error
}

func (s ssoFake) Authenticate(ctx context.Context, scopes []string) (*app.Token, error) {
	return s.token, s.err
}

func (s ssoFake) RefreshToken(ctx context.Context, refreshToken string) (*app.Token, error) {
	return s.token, s.err
}

func TestUpdateOrCreateCharacterFromSSO(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	test.NewTempApp(t)
	t.Run("create new character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		corporation := factory.CreateEveCorporation()
		factory.CreateEveEntityWithCategory(app.EveEntityCorporation, app.EveEntity{ID: corporation.ID})
		character := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			CorporationID: corporation.ID,
		})
		cs := characterservice.NewFake(st, characterservice.Params{
			SSOService: ssoFake{token: factory.CreateToken(app.Token{
				CharacterID:   character.ID,
				CharacterName: character.Name})},
		})
		var info string
		b := binding.BindString(&info)
		got, err := cs.UpdateOrCreateCharacterFromSSO(ctx, b)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, character.ID, got)
			ok, err := cs.HasCharacter(ctx, character.ID)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
			token, err := st.GetCharacterToken(ctx, character.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, token.CharacterID, character.ID)
			}
			x, err := st.GetCorporation(ctx, corporation.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, corporation, x.Corporation)
			}
		}
	})
	t.Run("update existing character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		corporation := factory.CreateEveCorporation()
		factory.CreateEveEntityWithCategory(app.EveEntityCorporation, app.EveEntity{ID: corporation.ID})
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			CorporationID: corporation.ID,
		})
		c := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
		factory.CreateCharacterToken(app.CharacterToken{
			AccessToken: "oldToken",
			CharacterID: c.ID,
		})
		cs := characterservice.NewFake(st, characterservice.Params{
			SSOService: ssoFake{token: factory.CreateToken(app.Token{
				CharacterID:   c.ID,
				CharacterName: c.EveCharacter.Name})},
		})
		var info string
		b := binding.BindString(&info)
		got, err := cs.UpdateOrCreateCharacterFromSSO(ctx, b)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c.ID, got)
			token, err := st.GetCharacterToken(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, token.CharacterID, c.ID)
			}
		}
	})
}

func TestTrainingWatchers(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("should enable watchers for characters with active queues only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c1.ID})
		c2 := factory.CreateCharacter()
		// when
		err := cs.EnableAllTrainingWatchers(ctx)
		// then
		if assert.NoError(t, err) {
			c1x, err := cs.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.True(t, c1x.IsTrainingWatched)
			}
			c2x, err := cs.GetCharacter(ctx, c2.ID)
			if assert.NoError(t, err) {
				assert.False(t, c2x.IsTrainingWatched)
			}
		}
	})
	t.Run("should disable all training watchers", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: true})
		c2 := factory.CreateCharacter()
		// when
		err := cs.DisableAllTrainingWatchers(ctx)
		// then
		if assert.NoError(t, err) {
			c1x, err := cs.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.False(t, c1x.IsTrainingWatched)
			}
			c2x, err := cs.GetCharacter(ctx, c2.ID)
			if assert.NoError(t, err) {
				assert.False(t, c2x.IsTrainingWatched)
			}
		}
	})
	t.Run("should enable watchers for character with active queues", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c1.ID})
		// when
		err := cs.EnableTrainingWatcher(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			c1a, err := cs.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.True(t, c1a.IsTrainingWatched)
			}
		}
	})
	t.Run("should not enable watchers for character without active queues", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		// when
		err := cs.EnableTrainingWatcher(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			c1a, err := cs.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.False(t, c1a.IsTrainingWatched)
			}
		}
	})
}

func TestNotifyUpdatedContracts(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	const characterID = 7
	earliest := time.Now().UTC().Add(-6 * time.Hour)
	now := time.Now().UTC()
	cases := []struct {
		name           string
		acceptorID     int32
		status         app.ContractStatus
		statusNotified app.ContractStatus
		typ            app.ContractType
		updatedAt      time.Time
		shouldNotify   bool
	}{
		{"notify new courier 1", 42, app.ContractStatusInProgress, app.ContractStatusUndefined, app.ContractTypeCourier, now, true},
		{"notify new courier 2", 42, app.ContractStatusFinished, app.ContractStatusUndefined, app.ContractTypeCourier, now, true},
		{"notify new courier 3", 42, app.ContractStatusFailed, app.ContractStatusUndefined, app.ContractTypeCourier, now, true},
		{"don't notify courier", 0, app.ContractStatusOutstanding, app.ContractStatusUndefined, app.ContractTypeCourier, now, false},
		{"notify new item exchange", 42, app.ContractStatusFinished, app.ContractStatusUndefined, app.ContractTypeItemExchange, now, true},
		{"don't notify again", 42, app.ContractStatusInProgress, app.ContractStatusInProgress, app.ContractTypeCourier, now, false},
		{"don't notify when acceptor is character", characterID, app.ContractStatusInProgress, app.ContractStatusUndefined, app.ContractTypeCourier, now, false},
		{"don't notify when contract is too old", 42, app.ContractStatusInProgress, app.ContractStatusUndefined, app.ContractTypeCourier, now.Add(-12 * time.Hour), false},
		{"don't notify item exchange", 0, app.ContractStatusOutstanding, app.ContractStatusUndefined, app.ContractTypeItemExchange, now, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.TruncateTables(db)
			if tc.acceptorID != 0 {
				factory.CreateEveEntityCharacter(app.EveEntity{ID: tc.acceptorID})
			}
			ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{ID: characterID})
			c := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
			o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
				AcceptorID:     tc.acceptorID,
				CharacterID:    c.ID,
				Status:         tc.status,
				StatusNotified: tc.statusNotified,
				Type:           tc.typ,
				UpdatedAt:      tc.updatedAt,
			})
			var sendCount int
			// when
			err := cs.NotifyUpdatedContracts(ctx, o.CharacterID, earliest, func(title string, content string) {
				sendCount++
			})
			// then
			if assert.NoError(t, err) {
				assert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}

func TestUpdateMail(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("Can fetch new mail", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c1 := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c1.ID})
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
		mailID := 7
		labelIDs := []int32{16, 32}
		timestamp := "2015-09-30T16:07:00Z"
		subject := "test"
		dataHeader := []map[string]any{
			{
				"from":       e1.ID,
				"is_read":    true,
				"labels":     labelIDs,
				"mail_id":    mailID,
				"recipients": recipients,
				"subject":    subject,
				"timestamp":  timestamp,
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/", c1.ID),
			httpmock.NewJsonResponderOrPanic(200, dataHeader),
		)
		dataMail := map[string]any{
			"body":       "blah blah blah",
			"from":       e1.ID,
			"labels":     labelIDs,
			"read":       true,
			"recipients": recipients,
			"subject":    "test",
			"timestamp":  timestamp,
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/%d/", c1.ID, mailID),
			httpmock.NewJsonResponderOrPanic(200, dataMail),
		)
		dataMailList := []map[string]any{
			{
				"mailing_list_id": m1.ID,
				"name":            "test_mailing_list",
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/lists/", c1.ID),
			httpmock.NewJsonResponderOrPanic(200, dataMailList),
		)
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
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/mail/labels/", c1.ID),
			httpmock.NewJsonResponderOrPanic(200, dataMailLabel),
		)
		// when
		_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c1.ID,
			Section:     app.SectionMailLabels,
		})
		if assert.NoError(t, err) {
			_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterUpdateSectionParams{
				CharacterID: c1.ID,
				Section:     app.SectionMailLists,
			})
			if assert.NoError(t, err) {
				_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterUpdateSectionParams{
					CharacterID: c1.ID,
					Section:     app.SectionMails,
				})
				// then
				if assert.NoError(t, err) {
					m, err := s.GetMail(ctx, c1.ID, int32(mailID))
					if assert.NoError(t, err) {
						assert.Equal(t, "blah blah blah", m.Body)
					}
					labels, err := st.ListCharacterMailLabelsOrdered(ctx, c1.ID)
					if assert.NoError(t, err) {
						got := set.Of[int32]()
						for _, l := range labels {
							got.Add(l.LabelID)
						}
						want := set.Of(labelIDs...)
						assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
					}
					lists, err := st.ListCharacterMailListsOrdered(ctx, c2.ID)
					if assert.NoError(t, err) {
						got := set.Of[int32]()
						for _, l := range lists {
							got.Add(l.ID)
						}
						want := set.Of(m2.ID)
						assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
					}
				}
			}
		}
	})
	t.Run("Can update existing mail", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		e1 := factory.CreateEveEntityCharacter()
		e2 := factory.CreateEveEntityCharacter()
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: 16})
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: 32}) // obsolete
		m1 := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList})
		timestamp, _ := time.Parse("2006-01-02T15:04:05.999MST", "2015-09-30T16:07:00Z")
		mailID := int32(7)
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			Body:         "blah blah blah",
			CharacterID:  c.ID,
			FromID:       e1.ID,
			LabelIDs:     []int32{16},
			MailID:       mailID,
			IsRead:       false,
			RecipientIDs: []int32{e2.ID, m1.ID},
			Subject:      "test",
			Timestamp:    timestamp,
		})

		dataMailList := []map[string]any{
			{
				"mailing_list_id": m1.ID,
				"name":            "test_mailing_list",
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/lists/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, dataMailList),
		)
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
			httpmock.NewJsonResponderOrPanic(200, dataMailLabel),
		)
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
		dataHeader := []map[string]any{
			{
				"from":       e1.ID,
				"is_read":    true,
				"labels":     []int{32},
				"mail_id":    mailID,
				"recipients": recipients,
				"subject":    "test",
				"timestamp":  "2015-09-30T16:07:00Z",
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, dataHeader),
		)
		// when
		_, err := s.UpdateSectionIfNeeded(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionMails,
		})
		// then
		if assert.NoError(t, err) {
			m, err := s.GetMail(ctx, c.ID, mailID)
			if assert.NoError(t, err) {
				assert.Equal(t, "blah blah blah", m.Body)
				assert.True(t, m.IsRead)
				assert.Len(t, m.Labels, 1)
				assert.Equal(t, int32(32), m.Labels[0].LabelID)
				assert.Len(t, m.Recipients, 2)
			}
			labels, err := st.ListCharacterMailLabelsOrdered(ctx, c.ID)
			if assert.NoError(t, err) {
				got := set.Of[int32]()
				for _, l := range labels {
					got.Add(l.LabelID)
				}
				want := set.Of[int32](16, 32)
				assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
			}
		}
	})
}

func TestNotifyMails(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	now := time.Now().UTC()
	earliest := now.Add(-12 * time.Hour)
	cases := []struct {
		name         string
		timestamp    time.Time
		isProcessed  bool
		shouldNotify bool
	}{
		{"send unprocessed", now, false, true},
		{"don't send processed", now, true, false},
		{"don't send old", now.Add(-16 * time.Hour), false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.TruncateTables(db)
			n := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
				IsProcessed: tc.isProcessed,
				Timestamp:   tc.timestamp,
			})
			var sendCount int
			// when
			err := cs.NotifyMails(ctx, n.CharacterID, earliest, func(title string, content string) {
				sendCount++
			})
			// then
			if assert.NoError(t, err) {
				assert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}

func TestSendMail(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := characterservice.NewFake(st)
	t.Run("Can send mail", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		r := factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/mail/", c.ID),
			httpmock.NewJsonResponderOrPanic(201, 123))

		// when
		mailID, err := s.SendMail(ctx, c.ID, "subject", []*app.EveEntity{r}, "body")
		// then
		if assert.NoError(t, err) {
			m, err := s.GetMail(ctx, c.ID, mailID)
			if assert.NoError(t, err) {
				assert.Equal(t, "body", m.Body)
			}
		}
	})
}

func TestNotifyCommunications(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	now := time.Now().UTC()
	earliest := now.Add(-12 * time.Hour)
	typesEnabled := set.Of(string(evenotification.StructureUnderAttack))
	cases := []struct {
		name         string
		typ          evenotification.Type
		timestamp    time.Time
		isProcessed  bool
		shouldNotify bool
	}{
		{"send unprocessed", evenotification.StructureUnderAttack, now, false, true},
		{"don't send old unprocessed", evenotification.StructureUnderAttack, now.Add(-16 * time.Hour), false, false},
		{"don't send not enabled types", evenotification.SkyhookOnline, now, false, false},
		{"don't resend already processed", evenotification.StructureUnderAttack, now, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.TruncateTables(db)
			n := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
				IsProcessed: tc.isProcessed,
				Title:       optional.From("title"),
				Body:        optional.From("body"),
				Type:        string(tc.typ),
				Timestamp:   tc.timestamp,
			})
			var sendCount int
			// when
			err := cs.NotifyCommunications(ctx, n.CharacterID, earliest, typesEnabled, func(title string, content string) {
				sendCount++
			})
			// then
			if assert.NoError(t, err) {
				assert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}

func TestCountNotificatios(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	// given
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	c := factory.CreateCharacter()
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		CharacterID: c.ID,
		Type:        string(evenotification.StructureDestroyed),
	})
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		CharacterID: c.ID,
		Type:        string(evenotification.MoonminingExtractionStarted),
	})
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		CharacterID: c.ID,
		Type:        string(evenotification.MoonminingExtractionStarted),
		IsRead:      true,
	})
	factory.CreateCharacterNotification()
	// when
	got, err := cs.CountNotifications(ctx, c.ID)
	if assert.NoError(t, err) {
		want := map[app.NotificationGroup][]int{
			app.GroupStructure:  {1, 1},
			app.GroupMoonMining: {2, 1},
		}
		assert.Equal(t, want, got)
	}
}

func TestNotifyExpiredExtractions(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	now := time.Now().UTC()
	earliest := now.Add(-24 * time.Hour)
	cases := []struct {
		name         string
		isExtractor  bool
		expiryTime   time.Time
		lastNotified time.Time
		shouldNotify bool
	}{
		{"extraction expired and not yet notified", true, now.Add(-3 * time.Hour), time.Time{}, true},
		{"extraction expired and already notified", true, now.Add(-3 * time.Hour), now.Add(-3 * time.Hour), false},
		{"extraction not expired", true, now.Add(3 * time.Hour), time.Time{}, false},
		{"extraction expired old", true, now.Add(-48 * time.Hour), time.Time{}, false},
		{"no expiration date", true, time.Time{}, time.Time{}, false},
		{"non extractor expired", false, now.Add(-3 * time.Hour), time.Time{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.TruncateTables(db)
			product := factory.CreateEveType()
			p := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
				LastNotified: tc.lastNotified,
			})
			if tc.isExtractor {
				factory.CreatePlanetPinExtractor(storage.CreatePlanetPinParams{
					CharacterPlanetID:      p.ID,
					ExpiryTime:             tc.expiryTime,
					ExtractorProductTypeID: optional.From(product.ID),
				})
			} else {
				factory.CreatePlanetPin(storage.CreatePlanetPinParams{
					CharacterPlanetID: p.ID,
					ExpiryTime:        tc.expiryTime,
				})
			}
			var sendCount int
			// when
			err := cs.NotifyExpiredExtractions(ctx, p.CharacterID, earliest, func(title string, content string) {
				sendCount++
			})
			// then
			if assert.NoError(t, err) {
				assert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}

func TestUpdateCharacterSection(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := characterservice.NewFake(st)
	section := app.SectionImplants
	ctx := context.Background()
	t.Run("should report true when changed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{et.ID}))
		// when
		changed, err := s.UpdateSectionIfNeeded(
			ctx, app.CharacterUpdateSectionParams{CharacterID: c.ID, Section: section})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.True(t, x.IsOK())
			}
		}
	})
	t.Run("should not update and report false when not changed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		data := []int32{100}
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
			CompletedAt: time.Now().Add(-6 * time.Hour),
			Data:        data,
		})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.UpdateSectionIfNeeded(
			ctx, app.CharacterUpdateSectionParams{CharacterID: c.ID, Section: section})
		// then
		if assert.NoError(t, err) {
			assert.False(t, changed)
			x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.CompletedAt, 5*time.Second)
			}
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			xx, err := st.ListCharacterImplants(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 0)
			}
		}
	})
	t.Run("should not fetch or update when not expired and report false", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{et.ID}))
		// when
		changed, err := s.UpdateSectionIfNeeded(
			ctx, app.CharacterUpdateSectionParams{
				CharacterID: c.ID,
				Section:     section,
			})
		// then
		if assert.NoError(t, err) {
			assert.False(t, changed)
			assert.Equal(t, 0, httpmock.GetTotalCallCount())
			xx, err := st.ListCharacterImplants(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 0)
			}
		}
	})
	t.Run("should record when update failed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(500, map[string]string{"error": "dummy error"}))
		// when
		_, err := s.UpdateSectionIfNeeded(
			ctx, app.CharacterUpdateSectionParams{CharacterID: c.ID, Section: section})
		// then
		if assert.Error(t, err) {
			x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.False(t, x.IsOK())
				assert.Equal(t, "500 Internal Server Error", x.ErrorMessage)
			}
		}
	})
	t.Run("should fetch and update when not expired and force update requested", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{et.ID}))
		// when
		_, err := s.UpdateSectionIfNeeded(
			ctx, app.CharacterUpdateSectionParams{
				CharacterID: c.ID,
				Section:     section,
				ForceUpdate: true,
			})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			xx, err := st.ListCharacterImplants(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 1)
			}
		}
	})
	t.Run("should update when not changed and force update requested", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		data := []int32{100}
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
			CompletedAt: time.Now().Add(-6 * time.Hour),
			Data:        data,
		})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		_, err := s.UpdateSectionIfNeeded(
			ctx, app.CharacterUpdateSectionParams{
				CharacterID: c.ID,
				Section:     section,
				ForceUpdate: true,
			})
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.CompletedAt, 5*time.Second)
			}
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			xx, err := st.ListCharacterImplants(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 1)
			}
		}
	})
}

func TestUpdateTickerNotifyExpiredTraining(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("send notification when watched & expired", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: true})
		var sendCount int
		// when
		err := cs.NotifyExpiredTraining(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, sendCount, 1)
		}
	})
	t.Run("do nothing when not watched", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		var sendCount int
		// when
		err := cs.NotifyExpiredTraining(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, sendCount, 0)
		}
	})
	t.Run("don't send notification when watched and training ongoing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: true})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		var sendCount int
		// when
		err := cs.NotifyExpiredTraining(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, sendCount, 0)
		}
	})
}

func TestDeleteCharacter(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("delete character and delete corporation when it has no members anymore", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		ec := factory.CreateEveCorporation()
		corporation := factory.CreateCorporation(ec.ID)
		factory.CreateEveEntityWithCategory(app.EveEntityCorporation, app.EveEntity{ID: ec.ID})
		x := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: ec.ID})
		character := factory.CreateCharacter(storage.CreateCharacterParams{ID: x.ID})
		// when
		err := cs.DeleteCharacter(ctx, character.ID)
		// then
		if assert.NoError(t, err) {
			_, err = st.GetCharacter(ctx, character.ID)
			assert.ErrorIs(t, err, app.ErrNotFound)
			_, err = st.GetCorporation(ctx, corporation.ID)
			assert.ErrorIs(t, err, app.ErrNotFound)
		}
	})
	t.Run("delete character and keep corporation when it still has members", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		ec := factory.CreateEveCorporation()
		corporation := factory.CreateCorporation(ec.ID)
		factory.CreateEveEntityWithCategory(app.EveEntityCorporation, app.EveEntity{ID: ec.ID})
		x1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: ec.ID})
		character := factory.CreateCharacter(storage.CreateCharacterParams{ID: x1.ID})
		x2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: ec.ID})
		factory.CreateCharacter(storage.CreateCharacterParams{ID: x2.ID})
		// when
		err := cs.DeleteCharacter(ctx, character.ID)
		// then
		if assert.NoError(t, err) {
			_, err = st.GetCharacter(ctx, character.ID)
			assert.ErrorIs(t, err, app.ErrNotFound)
			_, err = st.GetCorporation(ctx, corporation.ID)
			assert.NoError(t, err)
		}
	})
}
