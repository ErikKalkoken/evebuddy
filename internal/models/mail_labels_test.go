package models_test

import (
	"example/esiapp/internal/models"
	"example/esiapp/internal/set"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// createMailLabel is a test factory for MailLabel objects
func createMailLabel(args ...models.MailLabel) models.MailLabel {
	var l models.MailLabel
	if len(args) > 0 {
		l = args[0]
	}
	if l.Character.ID == 0 {
		l.Character = createCharacter()
	}
	if l.LabelID == 0 {
		ll, err := models.FetchAllMailLabels(l.Character.ID)
		if err != nil {
			panic(err)
		}
		var ids []int32
		for _, o := range ll {
			ids = append(ids, o.LabelID)
		}
		if len(ids) > 0 {
			l.LabelID = slices.Max(ids) + 1
		} else {
			l.LabelID = 1
		}
	}
	if l.Name == "" {
		l.Name = fmt.Sprintf("Generated name #%d", l.LabelID)
	}
	if l.Color == "" {
		l.Color = "#FFFFFF"
	}
	if l.UnreadCount == 0 {
		l.UnreadCount = int32(rand.IntN(1000))
	}
	if err := l.Save(); err != nil {
		panic(err)
	}
	return l
}

func TestMailLabelSaveNew(t *testing.T) {
	// given
	models.TruncateTables()
	c := createCharacter()
	l := models.MailLabel{
		Character:   c,
		Color:       "xyz",
		LabelID:     1,
		Name:        "Dummy",
		UnreadCount: 42,
	}
	// when
	err := l.Save()
	// then
	if assert.NoError(t, err) {
		l2, err := models.FetchMailLabel(c.ID, l.LabelID)
		if assert.NoError(t, err) {
			assert.Equal(t, l.Name, l2.Name)
		}
	}
}

func TestMailLabelCanFetchAllLabelsReturnsSlice(t *testing.T) {
	// given
	models.TruncateTables()
	l := createMailLabel()
	// when
	l2, err := models.FetchMailLabel(l.Character.ID, l.LabelID)
	if assert.NoError(t, err) {
		assert.Equal(t, l.Name, l2.Name)
	}
}

func TestMailLabelCanFetchAllLabels(t *testing.T) {
	// given
	models.TruncateTables()
	c := createCharacter()
	createMailLabel(models.MailLabel{Character: c, LabelID: 3})
	createMailLabel(models.MailLabel{Character: c, LabelID: 7})
	// when
	ll, err := models.FetchAllMailLabels(c.ID)
	if assert.NoError(t, err) {
		gotIDs := set.New[int32]()
		for _, l := range ll {
			gotIDs.Add(l.LabelID)
		}
		wantIDs := set.NewFromSlice([]int32{3, 7})
		assert.Equal(t, wantIDs, gotIDs)
	}
}

func TesFetchAllMailLabelsReturnsEmptySliceWhenNoRows(t *testing.T) {
	// given
	models.TruncateTables()
	c := createCharacter()
	// when
	ll, err := models.FetchAllMailLabels(c.ID)
	if assert.NoError(t, err) {
		assert.Equal(t, 0, len(ll))
	}
}
