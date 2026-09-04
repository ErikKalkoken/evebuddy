package evenotification_test

import (
	"testing"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestGameTime_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	t.Run("GameTimeAdded", func(t *testing.T) {
		title, body, err := en.RenderESI(t.Context(), app.GameTimeAdded, optional.New("{}"), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Game time added", title)
		assert.NotEmpty(t, body)
	})

	t.Run("GameTimeReceived", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		sender := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"message":      "Enjoy!",
			"offerID":      701,
			"quantity":     5,
			"senderCharID": sender.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.GameTimeReceived, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Game time received from "+sender.Name, title)
		assert.Contains(t, body, "5")
		assert.Contains(t, body, "Enjoy!")
	})

	t.Run("GameTimeSent", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		sender := f.CreateEveEntityCharacter()
		receiver := f.CreateEveEntityCharacter()
		text, err := yaml.Marshal(map[string]any{
			"receiverCharID": receiver.ID,
			"senderCharID":   sender.ID,
		})
		require.NoError(t, err)

		title, body, err := en.RenderESI(t.Context(), app.GameTimeSent, optional.New(string(text)), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Game time sent to "+receiver.Name, title)
		assert.Contains(t, body, sender.Name)
		assert.Contains(t, body, receiver.Name)
	})
}
