package characterservice_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestWriteNotificationTypeFixtures(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	cs := characterservice.NewFake(characterservice.Params{Storage: st})
	now := time.Now().UTC()

	sender := factory.CreateEveEntityCharacter()
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		Type:      "CorpAppNewMsg",
		SenderID:  sender.ID,
		Timestamp: now.Add(-time.Hour), // older, should be replaced by the newer one below
		Text:      optional.New("applicationText: example\ncharID: 1011\ncorpID: 2001\n"),
		IsRead:    false,
	})
	newest := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		Type:      "CorpAppNewMsg",
		SenderID:  sender.ID,
		Timestamp: now,
		Text:      optional.New("applicationText: example\ncharID: 1011\nallyCharID: 1011\ncorpID: 2001\n"),
		IsRead:    true,
	})
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		Type:      "CharAppAcceptMsg",
		SenderID:  factory.CreateEveEntityCorporation().ID,
		Timestamp: now,
		Text:      optional.New("charID: 1011\ncorpID: 2001\n"),
	})
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		Type:      "NotAKnownESIType",
		Timestamp: now,
	})

	var buf bytes.Buffer
	err := cs.WriteNotificationTypeFixtures(t.Context(), &buf)
	require.NoError(t, err)

	var got []struct {
		NotificationID int64     `json:"notification_id"`
		Type           string    `json:"type"`
		SenderID       int64     `json:"sender_id"`
		SenderType     string    `json:"sender_type"`
		Timestamp      time.Time `json:"timestamp"`
		Text           string    `json:"text"`
		IsRead         bool      `json:"is_read"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))

	// one fixture per known type, unknown type excluded
	require.Len(t, got, 2)
	types := []string{got[0].Type, got[1].Type}
	assert.ElementsMatch(t, []string{"CorpAppNewMsg", "CharAppAcceptMsg"}, types)

	var corpAppNewMsg struct {
		NotificationID int64
		Type           string
		SenderID       int64
		SenderType     string
		Timestamp      time.Time
		Text           string
		IsRead         bool
	}
	for _, x := range got {
		if x.Type == "CorpAppNewMsg" {
			corpAppNewMsg.NotificationID = x.NotificationID
			corpAppNewMsg.SenderID = x.SenderID
			corpAppNewMsg.SenderType = x.SenderType
			corpAppNewMsg.Text = x.Text
			corpAppNewMsg.IsRead = x.IsRead
		}
	}

	// the newest notification for the type was picked, not the older one
	assert.Equal(t, newest.NotificationID, corpAppNewMsg.NotificationID)
	assert.True(t, corpAppNewMsg.IsRead)
	assert.Equal(t, "character", corpAppNewMsg.SenderType)

	// sender ID and IDs in text are anonymized
	assert.NotEqual(t, sender.ID, corpAppNewMsg.SenderID)
	assert.NotContains(t, corpAppNewMsg.Text, "1011")
	assert.NotContains(t, corpAppNewMsg.Text, "2001")

	// the same original ID (charID and allyCharID both 1011) maps to the same placeholder
	var data map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(corpAppNewMsg.Text), &data))
	charID, ok := data["charID"]
	require.True(t, ok)
	allyCharID, ok := data["allyCharID"]
	require.True(t, ok)
	assert.Equal(t, charID, allyCharID)
	assert.NotEqual(t, int64(1011), charID)
}

func TestWriteNotificationTypeFixtures_AnonymizesStrings(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	cs := characterservice.NewFake(characterservice.Params{Storage: st})

	text := "" +
		"corpName: Wayne Enterprises\n" +
		"corpLink: <a href=\"showinfo:2//2001\">Wayne Enterprises</a>\n" +
		"corpLinkData:\n" +
		"- showinfo\n" +
		"- 2\n" +
		"- 2001\n" +
		"firedByLink: <a href=\"showinfo:1376//1001\">Bruce Wayne</a>\n" +
		"applicationText: please let me join your corporation\n"
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		Type: "CorpAppNewMsg",
		Text: optional.New(text),
	})

	var buf bytes.Buffer
	err := cs.WriteNotificationTypeFixtures(t.Context(), &buf)
	require.NoError(t, err)

	var got []struct {
		Text string `json:"text"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got, 1)
	anonymized := got[0].Text

	var data map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(anonymized), &data))

	// the "showinfo" literal marker in *LinkData arrays must be preserved
	linkData, ok := data["corpLinkData"].([]any)
	require.True(t, ok)
	assert.Equal(t, "showinfo", linkData[0])

	// name links keep their href structure but get a new display name
	corpLink, ok := data["corpLink"].(string)
	require.True(t, ok)
	assert.Regexp(t, `^<a href="showinfo:2//2001">.+</a>$`, corpLink)
	assert.NotContains(t, corpLink, "Wayne Enterprises")

	firedByLink, ok := data["firedByLink"].(string)
	require.True(t, ok)
	assert.Regexp(t, `^<a href="showinfo:1376//1001">.+</a>$`, firedByLink)
	assert.NotContains(t, firedByLink, "Bruce Wayne")

	// generic free text fields are replaced too
	corpName, ok := data["corpName"].(string)
	require.True(t, ok)
	assert.NotEqual(t, "Wayne Enterprises", corpName)
	applicationText, ok := data["applicationText"].(string)
	require.True(t, ok)
	assert.NotEqual(t, "please let me join your corporation", applicationText)
}
