package characterservice

import (
	"fmt"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateNotificationESI_CreateRendered(t *testing.T) {
	db, st, f := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// given
	testutil.MustTruncateTables(db)
	httpmock.Reset()
	const (
		title          = "title"
		body           = "body"
		nt             = app.CharLeftCorpMsg
		notificationID = 42
		isRead         = true
	)
	timestamp := f.RandomTimeRounded()
	ens := &EVENotificationServiceStub{
		title: title,
		body:  body,
	}
	s := NewFake(Params{Storage: st, EveNotificationService: ens})
	c := f.CreateCharacter()
	f.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
	f.CreateEveEntityCharacter(*c.EveCharacter.ToEveEntity())
	corp := f.CreateEveEntityCorporation()
	char := f.CreateEveEntityCharacter()
	b, err := yaml.Marshal(map[string]any{
		"charID": char.ID,
		"corpID": corp.ID,
	})
	require.NoError(t, err)
	text := string(b)

	data := []map[string]any{{
		"is_read":         isRead,
		"notification_id": notificationID,
		"sender_id":       char.ID,
		"sender_type":     "character",
		"text":            text,
		"timestamp":       timestamp.Format(time.RFC3339),
		"type":            nt.String(),
	}}
	httpmock.RegisterResponder(
		"GET",
		fmt.Sprintf("https://esi.evetech.net/characters/%d/notifications", c.ID),
		httpmock.NewJsonResponderOrPanic(200, data),
	)

	// when
	changed, err := s.updateNotificationsESI(t.Context(), characterSectionUpdateParams{
		characterID: c.ID,
		section:     app.SectionCharacterNotifications,
	})

	// then
	require.NoError(t, err)
	assert.True(t, changed)
	o, err := st.GetCharacterNotification(t.Context(), c.ID, notificationID)
	require.NoError(t, err)
	xassert.Equal(t, isRead, o.IsRead)
	xassert.Equal(t, nt, o.Type)
	xassert.Equal(t, char, o.Sender)
	xassert.EqualOptional(t, title, o.Title)
	xassert.EqualOptional(t, body, o.Body)
	xassert.EqualOptional(t, text, o.Text)
	xassert.Equal(t, timestamp, o.Timestamp)
	xassert.EqualOptional(t, c.EveCharacter.Corporation, o.Recipient)

	ids, err := st.ListCharacterNotificationIDs(t.Context(), c.ID)
	require.NoError(t, err)
	xassert.Equal(t, set.Of[int64](notificationID), ids)
}

func TestUpdateNotificationESI_CreateUnRendered(t *testing.T) {
	db, st, f := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// given
	testutil.MustTruncateTables(db)
	httpmock.Reset()
	const (
		nt             = app.KillReportVictim
		notificationID = 42
		isRead         = true
	)
	timestamp := f.RandomTimeRounded()
	ens := &EVENotificationServiceStub{
		err: app.ErrNotFound,
	}
	s := NewFake(Params{Storage: st, EveNotificationService: ens})
	c := f.CreateCharacter()
	f.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
	f.CreateEveEntityCharacter(*c.EveCharacter.ToEveEntity())
	sender := f.CreateEveEntityCorporation()
	shiptype := f.CreateEveType()
	b, err := yaml.Marshal(map[string]any{
		"killMailHash":     "d2eaf0c773d2bf5b41849895321ced7b223f1edf",
		"killMailID":       98973153,
		"victimShipTypeID": shiptype.ID,
	})
	require.NoError(t, err)
	text := string(b)

	data := []map[string]any{{
		"is_read":         isRead,
		"notification_id": notificationID,
		"sender_id":       sender.ID,
		"sender_type":     "corporation",
		"text":            text,
		"timestamp":       timestamp.Format(time.RFC3339),
		"type":            nt.String(),
	}}
	httpmock.RegisterResponder(
		"GET",
		fmt.Sprintf("https://esi.evetech.net/characters/%d/notifications", c.ID),
		httpmock.NewJsonResponderOrPanic(200, data),
	)

	// when
	changed, err := s.updateNotificationsESI(t.Context(), characterSectionUpdateParams{
		characterID: c.ID,
		section:     app.SectionCharacterNotifications,
	})

	// then
	require.NoError(t, err)
	assert.True(t, changed)
	o, err := st.GetCharacterNotification(t.Context(), c.ID, notificationID)
	require.NoError(t, err)
	xassert.Equal(t, isRead, o.IsRead)
	xassert.Equal(t, nt, o.Type)
	xassert.Equal(t, sender, o.Sender)
	xassert.Empty(t, o.Title)
	xassert.Empty(t, o.Body)
	xassert.EqualOptional(t, text, o.Text)
	xassert.Equal(t, timestamp, o.Timestamp)
	xassert.EqualOptional(t, c.EveCharacter.ToEveEntity(), o.Recipient)

	ids, err := st.ListCharacterNotificationIDs(t.Context(), c.ID)
	require.NoError(t, err)
	xassert.Equal(t, set.Of[int64](notificationID), ids)
}

func TestUpdateNotificationESI_Update(t *testing.T) {
	db, st, f := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	cases := []struct {
		name string

		oldBody   string
		oldTitle  string
		oldIsRead bool

		newBody   string
		newTitle  string
		newIsRead bool

		forceUpdate bool

		wantBody    string
		wantTitle   string
		wantIsRead  bool
		wantChanged bool
	}{
		{
			name:        "no update when same",
			oldBody:     "body",
			oldTitle:    "title",
			oldIsRead:   false,
			newBody:     "body",
			newTitle:    "title",
			newIsRead:   false,
			forceUpdate: false,
			wantChanged: false,
			wantBody:    "body",
			wantTitle:   "title",
			wantIsRead:  false,
		},
		{
			name:        "notif was read",
			oldBody:     "body",
			oldTitle:    "title",
			oldIsRead:   false,
			newBody:     "body",
			newTitle:    "title",
			newIsRead:   true,
			forceUpdate: false,
			wantChanged: true,
			wantBody:    "body",
			wantTitle:   "title",
			wantIsRead:  true,
		},
		{
			name:        "title changed",
			oldBody:     "body",
			oldTitle:    "title",
			oldIsRead:   false,
			newBody:     "body",
			newTitle:    "title2",
			newIsRead:   false,
			forceUpdate: false,
			wantChanged: true,
			wantBody:    "body",
			wantTitle:   "title2",
			wantIsRead:  false,
		},
		{
			name:        "body changed",
			oldBody:     "body",
			oldTitle:    "title",
			oldIsRead:   false,
			newBody:     "body2",
			newTitle:    "title",
			newIsRead:   false,
			forceUpdate: false,
			wantChanged: true,
			wantBody:    "body2",
			wantTitle:   "title",
			wantIsRead:  false,
		},
		{
			name:        "ignore local read",
			oldBody:     "body",
			oldTitle:    "title",
			oldIsRead:   true,
			newBody:     "body",
			newTitle:    "title",
			newIsRead:   false,
			forceUpdate: false,
			wantChanged: false,
			wantBody:    "body",
			wantTitle:   "title",
			wantIsRead:  true,
		},
		{
			name:        "keep local read when updated",
			oldBody:     "body",
			oldTitle:    "title",
			oldIsRead:   true,
			newBody:     "body2",
			newTitle:    "title",
			newIsRead:   false,
			forceUpdate: false,
			wantChanged: true,
			wantBody:    "body2",
			wantTitle:   "title",
			wantIsRead:  true,
		},
		{
			name:        "always update when forced",
			oldBody:     "body",
			oldTitle:    "title",
			oldIsRead:   false,
			newBody:     "body",
			newTitle:    "title",
			newIsRead:   false,
			forceUpdate: true,
			wantChanged: true,
			wantBody:    "body",
			wantTitle:   "title",
			wantIsRead:  false,
		},
		{
			name:        "update local read when force update",
			oldBody:     "body",
			oldTitle:    "title",
			oldIsRead:   true,
			newBody:     "body",
			newTitle:    "title",
			newIsRead:   false,
			forceUpdate: true,
			wantChanged: true,
			wantBody:    "body",
			wantTitle:   "title",
			wantIsRead:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.MustTruncateTables(db)
			httpmock.Reset()
			ens := &EVENotificationServiceStub{
				title: tc.newTitle,
				body:  tc.newBody,
			}
			s := NewFake(Params{Storage: st, EveNotificationService: ens})
			c := f.CreateCharacter()
			f.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
			f.CreateEveEntityCharacter(*c.EveCharacter.ToEveEntity())
			sender := f.CreateEveEntityCorporation()
			n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
				CharacterID: c.ID,
				Title:       optional.New(tc.oldTitle),
				Body:        optional.New(tc.oldBody),
				IsRead:      tc.oldIsRead,
			})
			data := []map[string]any{{
				"is_read":         tc.newIsRead,
				"notification_id": n1.NotificationID,
				"sender_id":       sender.ID,
				"sender_type":     "corporation",
				"text":            n1.Text,
				"timestamp":       n1.Timestamp.Format(time.RFC3339),
				"type":            n1.Type.String(),
			}}
			httpmock.RegisterResponder(
				"GET",
				fmt.Sprintf("https://esi.evetech.net/characters/%d/notifications", c.ID),
				httpmock.NewJsonResponderOrPanic(200, data),
			)

			// when
			changed, err := s.updateNotificationsESI(t.Context(), characterSectionUpdateParams{
				characterID: c.ID,
				section:     app.SectionCharacterNotifications,
				forceUpdate: tc.forceUpdate,
			})

			// then
			require.NoError(t, err)
			xassert.Equal(t, tc.wantChanged, changed)
			if tc.wantChanged {
				n2, err := st.GetCharacterNotification(t.Context(), c.ID, n1.NotificationID)
				require.NoError(t, err)
				xassert.Equal(t, tc.wantIsRead, n2.IsRead)
				xassert.EqualOptional(t, tc.wantTitle, n2.Title)
				xassert.EqualOptional(t, tc.wantBody, n2.Body)
			}
		})
	}
}

func TestUpdateNotificationESI_Other(t *testing.T) {
	db, st, f := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})

	t.Run("should add new and remove deleted notification", func(t *testing.T) {
		// given
		const newNotificationID = 9942
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := f.CreateCharacter()
		f.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		f.CreateEveEntityCharacter(*c.EveCharacter.ToEveEntity())
		n1 := f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID})
		f.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: c.ID}) // this should be removed
		sender := f.CreateEveEntityCorporation()
		timestamp := time.Now().Round(time.Second)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/notifications", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"notification_id": n1.NotificationID,
				"sender_id":       n1.Sender.ID,
				"sender_type":     n1.Sender.Category.String(),
				"text":            n1.Text.ValueOrZero(),
				"timestamp":       n1.Timestamp.Format(time.RFC3339),
				"type":            n1.Type.String(),
			}, {
				"notification_id": newNotificationID,
				"sender_id":       sender.ID,
				"sender_type":     "corporation",
				"text":            "",
				"timestamp":       timestamp.Format(time.RFC3339),
				"type":            "CorpNoLongerWarEligible",
			}}),
		)

		// when
		changed, err := s.updateNotificationsESI(t.Context(), characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterNotifications,
		})

		// then
		require.NoError(t, err)
		assert.True(t, changed)

		ids, err := st.ListCharacterNotificationIDs(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(newNotificationID, n1.NotificationID), ids)

		o, err := st.GetCharacterNotification(t.Context(), c.ID, newNotificationID)
		require.NoError(t, err)
		xassert.Equal(t, newNotificationID, o.NotificationID)
		assert.False(t, o.IsRead)
		xassert.Equal(t, sender, o.Sender)
		xassert.Equal(t, app.CorpNoLongerWarEligible, o.Type)
		xassert.Empty(t, o.Text)
		xassert.Equal(t, timestamp, o.Timestamp)
		xassert.EqualOptional(t, c.EveCharacter.Corporation, o.Recipient) // this is a corp notification
	})

	t.Run("should abort when sender can not be resolved", func(t *testing.T) {
		// given
		const (
			notificationID  = 1000000201
			text            = "amount: 3731016.4000000004\\nitemID: 1024881021663\\npayout: 1\\n"
			invalidSenderID = 666
		)
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := f.CreateCharacter()
		f.CreateEveEntityCharacter(*c.EveCharacter.ToEveEntity())
		f.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/notifications", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_read":         true,
				"notification_id": notificationID,
				"sender_id":       invalidSenderID,
				"sender_type":     "corporation",
				"text":            text,
				"timestamp":       "2017-08-16T10:08:00Z",
				"type":            "InsurancePayoutMsg",
			}}),
		)
		httpmock.RegisterResponder("POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewErrorResponder(fmt.Errorf("failed")),
		)
		// when
		_, err := s.updateNotificationsESI(t.Context(), characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterNotifications,
		})
		// then
		assert.Error(t, err)
		ids, err := st.ListCharacterNotificationIDs(t.Context(), c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](), ids)
	})

	t.Run("should not abort when entities inside notification can not be resolved", func(t *testing.T) {
		// given
		const (
			notificationID = 1000000201
			recruitID      = 666
		)
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := f.CreateCharacter()
		f.CreateEveEntityCharacter(*c.EveCharacter.ToEveEntity())
		f.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		corporation := c.EveCharacter.Corporation
		text := fmt.Sprintf("charID: %d\ncorpID: %d\n", recruitID, corporation.ID)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/notifications", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"is_read":         true,
				"notification_id": notificationID,
				"sender_id":       corporation.ID,
				"sender_type":     "corporation",
				"text":            text,
				"timestamp":       "2017-08-16T10:08:00Z",
				"type":            "CharLeftCorpMsg",
			}}),
		)
		httpmock.RegisterResponder("POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewErrorResponder(fmt.Errorf("failed")),
		)
		// when
		_, err := s.updateNotificationsESI(t.Context(), characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterNotifications,
		})
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterNotification(t.Context(), c.ID, notificationID)
		require.NoError(t, err)
		assert.True(t, o.IsRead)
		xassert.Equal(t, notificationID, o.NotificationID)
		xassert.Equal(t, corporation, o.Sender)
		xassert.Equal(t, app.CharLeftCorpMsg, o.Type)
		xassert.EqualOptional(t, text, o.Text)
		xassert.Equal(t, time.Date(2017, 8, 16, 10, 8, 0, 0, time.UTC), o.Timestamp)
		xassert.EqualOptional(t, corporation, o.Recipient)
	})
}
