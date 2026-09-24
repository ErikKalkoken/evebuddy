package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
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
		cn := &app.CharacterNotification{ID: 2}
		a.ReadingPane.currentNotification = cn
		a.MessagePane.syncSelection(id2idx)
		assert.Equal(t, 0, selectedCount)
		assert.Same(t, cn, a.ReadingPane.currentNotification)
	})
	t.Run("clears reading pane when current notification is gone", func(t *testing.T) {
		a := setup(t)
		a.ReadingPane.currentNotification = &app.CharacterNotification{ID: 99}
		a.MessagePane.syncSelection(id2idx)
		assert.Nil(t, a.ReadingPane.currentNotification)
	})
}
