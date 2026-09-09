package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCharacterMailLabelIDs(t *testing.T) {
	cm := app.CharacterMail{
		Labels: []*app.CharacterMailLabel{
			{LabelID: 1},
			{LabelID: 2},
		},
	}
	xassert.Equal(t, []int64{1, 2}, cm.LabelIDs())
}

func TestCharacterMailBodyPlain(t *testing.T) {
	cm := app.CharacterMail{Body: optional.New("alpha<br>bravo")}
	xassert.Equal(t, "alpha\nbravo", cm.BodyPlain())
}

func TestCharacterMailHeader(t *testing.T) {
	ts := time.Date(2024, 1, 2, 15, 4, 0, 0, time.UTC)
	cm := app.CharacterMail{
		From:      &app.EveEntity{Name: "Alice"},
		Timestamp: ts,
		Recipients: []*app.EveEntity{
			{Name: "Bob"},
			{Name: "Carol"},
		},
	}
	want := "From: Alice\n" +
		"Sent: 2024.01.02 15:04\n" +
		"To: Bob, Carol"
	xassert.Equal(t, want, cm.Header())
}

func TestCharacterMailString(t *testing.T) {
	ts := time.Date(2024, 1, 2, 15, 4, 0, 0, time.UTC)
	cm := app.CharacterMail{
		From:      &app.EveEntity{Name: "Alice"},
		Timestamp: ts,
		Subject:   optional.New("Hello"),
		Body:      optional.New("Body text"),
	}
	got := cm.String()
	assert.Contains(t, got, "Hello")
	assert.Contains(t, got, "From: Alice")
	assert.Contains(t, got, "Body text")
}

func TestCharacterMailRecipientNames(t *testing.T) {
	cm := app.CharacterMail{
		Recipients: []*app.EveEntity{
			{Name: "Bob"},
			{Name: "Carol"},
		},
	}
	xassert.Equal(t, []string{"Bob", "Carol"}, cm.RecipientNames())
}

func TestCharacterMailBodyToMarkdown(t *testing.T) {
	cm := app.CharacterMail{Body: optional.New("<b>bold</b>")}
	got := cm.BodyToMarkdown()
	assert.Contains(t, got, "bold")
}
