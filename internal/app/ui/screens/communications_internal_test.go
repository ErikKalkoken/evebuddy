package screens

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestParseIDs(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedID1 int64
		expectedID2 int64
		wantErr     bool
	}{
		{
			name:        "single ID without separator",
			input:       "12345",
			expectedID1: 12345,
			expectedID2: 0,
			wantErr:     false,
		},
		{
			name:        "two IDs with separator",
			input:       "12345//6789",
			expectedID1: 12345,
			expectedID2: 6789,
			wantErr:     false,
		},
		{
			name:        "zero values",
			input:       "0//0",
			expectedID1: 0,
			expectedID2: 0,
			wantErr:     false,
		},
		{
			name:    "invalid first ID",
			input:   "abc//6789",
			wantErr: true,
		},
		{
			name:    "invalid second ID",
			input:   "12345//xyz",
			wantErr: true,
		},
		{
			name:    "missing second ID after separator",
			input:   "12345//",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "integer overflow for int64",
			input:   "9223372036854775808", // MaxInt64 + 1
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id1, id2, err := parseIDs(tt.input)

			if tt.wantErr {
				// Assert that an error occurred and IDs return zero values on failure
				require.Error(t, err)
				assert.Equal(t, int64(0), id1)
				assert.Equal(t, int64(0), id2)
			} else {
				// Assert that no error occurred and IDs match expectations
				require.NoError(t, err)
				assert.Equal(t, tt.expectedID1, id1)
				assert.Equal(t, tt.expectedID2, id2)
			}
		})
	}
}

func TestCommunicationsMessagePane_SyncSelection(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	setup := func(t *testing.T) *Communications {
		a := NewCommunicationsForCharacter(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.MessagePane.rowsFiltered = []notificationRow{{id: 1}, {id: 2}, {id: 3}}
		a.MessagePane.messageList.Refresh()
		return a
	}
	id2idx := map[int64]int{1: 0, 2: 1, 3: 2}

	t.Run("mobile keeps current notification without selecting it again", func(t *testing.T) {
		a := setup(t)
		var selectedCount int
		a.MessagePane.OnSelected = func() {
			selectedCount++
		}
		a.ReadingPane.requestedID = 2
		a.MessagePane.syncSelection(id2idx)
		assert.Equal(t, 0, selectedCount)
		assert.EqualValues(t, 2, a.ReadingPane.requestedID)
	})
	t.Run("mobile does not scroll to current notification", func(t *testing.T) {
		a := setup(t)
		a.MessagePane.OnSelected = func() {}
		sender := &app.EveEntity{ID: 1, Name: "Sender", Category: app.EveEntityCorporation}
		var rows []notificationRow
		id2idx := make(map[int64]int)
		for i := range 50 {
			id := int64(i + 1)
			rows = append(rows, notificationRow{id: id, sender: sender, subject: "Subject"})
			id2idx[id] = i
		}
		a.MessagePane.rowsFiltered = rows
		test.WidgetRenderer(a.MessagePane.messageList)
		a.MessagePane.messageList.Resize(fyne.NewSize(300, 200))
		a.ReadingPane.requestedID = 40 // off-screen
		a.MessagePane.syncSelection(id2idx)
		assert.Zero(t, a.MessagePane.messageList.GetScrollOffset())
	})
	t.Run("clears reading pane when current notification is gone", func(t *testing.T) {
		a := setup(t)
		a.ReadingPane.requestedID = 99
		a.MessagePane.syncSelection(id2idx)
		assert.Zero(t, a.ReadingPane.requestedID)
	})
}

func TestCommunicationsReadingPane_LoadNotification(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	character := factory.CreateCharacterFull()
	n1 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: character.ID})
	n2 := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: character.ID})
	a := NewCommunicationsForCharacter(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	}))
	p := a.ReadingPane
	makeRow := func(n *app.CharacterNotification) notificationRow {
		return notificationRow{
			characterID:    n.CharacterID,
			id:             n.ID,
			notificationID: n.NotificationID,
			recipient:      n.Sender,
		}
	}
	r1, r2 := makeRow(n1), makeRow(n2)
	// request mimics set without starting the async load,
	// so tests can control the order in which loads complete.
	request := func(r notificationRow) {
		p.requestedID = r.id
	}

	t.Run("shows requested notification", func(t *testing.T) {
		p.clear()
		request(r1)
		p.loadNotification(t.Context(), r1)
		require.NotNil(t, p.currentNotification)
		assert.Equal(t, n1.ID, p.currentNotification.ID)
	})
	t.Run("ignores earlier request completing after later one", func(t *testing.T) {
		p.clear()
		request(r1)
		request(r2)
		p.loadNotification(t.Context(), r2)
		p.loadNotification(t.Context(), r1)
		require.NotNil(t, p.currentNotification)
		assert.Equal(t, n2.ID, p.currentNotification.ID)
	})
	t.Run("ignores earlier request completing before later one", func(t *testing.T) {
		p.clear()
		request(r1)
		request(r2)
		p.loadNotification(t.Context(), r1)
		assert.Nil(t, p.currentNotification)
		p.loadNotification(t.Context(), r2)
		require.NotNil(t, p.currentNotification)
		assert.Equal(t, n2.ID, p.currentNotification.ID)
	})
	t.Run("ignores result after pane was cleared", func(t *testing.T) {
		p.clear()
		request(r1)
		p.clear()
		p.loadNotification(t.Context(), r1)
		assert.Nil(t, p.currentNotification)
	})
	t.Run("keeps pending request when rows refresh before load completes", func(t *testing.T) {
		p.clear()
		a.MessagePane.rowsFiltered = []notificationRow{r1, r2}
		request(r1)
		a.MessagePane.syncSelection(map[int64]int{r1.id: 0, r2.id: 1})
		p.loadNotification(t.Context(), r1)
		require.NotNil(t, p.currentNotification)
		assert.Equal(t, n1.ID, p.currentNotification.ID)
	})
	t.Run("shows error when notification can not be loaded", func(t *testing.T) {
		p.clear()
		r := notificationRow{
			characterID:    character.ID,
			id:             999_999_999,
			notificationID: 999_999_999, // does not exist
		}
		request(r)
		p.loadNotification(t.Context(), r)
		assert.Nil(t, p.currentNotification)
		assert.Contains(t, p.bodyText.String(), "ERROR")
	})
}
