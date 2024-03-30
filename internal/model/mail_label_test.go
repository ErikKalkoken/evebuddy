package model_test

import (
	"example/esiapp/internal/helper/set"
	"example/esiapp/internal/model"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// createMailLabel is a test factory for MailLabel objects
func createMailLabel(args ...model.MailLabel) model.MailLabel {
	var l model.MailLabel
	if len(args) > 0 {
		l = args[0]
	}
	if l.Character.ID == 0 {
		l.Character = createCharacter()
	}
	if l.LabelID == 0 {
		ll, err := model.FetchAllMailLabels(l.Character.ID)
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
			l.LabelID = 100
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
	model.TruncateTables()
	c := createCharacter()
	l := model.MailLabel{
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
		l2, err := model.FetchMailLabel(c.ID, l.LabelID)
		if assert.NoError(t, err) {
			assert.Equal(t, l.Name, l2.Name)
		}
	}
}

func TestMailLabelShouldReturnErrorWhenNoCharacter(t *testing.T) {
	// given
	model.TruncateTables()
	l := model.MailLabel{
		Color:       "xyz",
		LabelID:     1,
		Name:        "Dummy",
		UnreadCount: 42,
	}
	// when
	err := l.Save()
	// then
	assert.Error(t, err)
}
func TestMailLabelCanFetchAllLabelsReturnsSlice(t *testing.T) {
	// given
	model.TruncateTables()
	l := createMailLabel()
	// when
	l2, err := model.FetchMailLabel(l.Character.ID, l.LabelID)
	if assert.NoError(t, err) {
		assert.Equal(t, l.Name, l2.Name)
	}
}

func TestCanFetchAllMailLabelsForCharacter(t *testing.T) {
	// given
	model.TruncateTables()
	c1 := createCharacter()
	l1 := createMailLabel(model.MailLabel{Character: c1, LabelID: 103})
	fmt.Println(l1)
	l2 := createMailLabel(model.MailLabel{Character: c1, LabelID: 107})
	fmt.Println(l2)
	c2 := createCharacter()
	createMailLabel(model.MailLabel{Character: c2, LabelID: 113})
	// when
	ll, err := model.FetchAllMailLabels(c1.ID)
	if assert.NoError(t, err) {
		gotIDs := set.New[int32]()
		for _, l := range ll {
			gotIDs.Add(l.LabelID)
		}
		wantIDs := set.NewFromSlice([]int32{103, 107})
		assert.Equal(t, wantIDs, gotIDs)
	}
}

func TesFetchAllMailLabelsReturnsEmptySliceWhenNoRows(t *testing.T) {
	// given
	model.TruncateTables()
	c := createCharacter()
	// when
	ll, err := model.FetchAllMailLabels(c.ID)
	if assert.NoError(t, err) {
		assert.Empty(t, ll)
	}
}

func TestFetchMailLabels(t *testing.T) {
	// given
	model.TruncateTables()
	c := createCharacter()
	createMailLabel(model.MailLabel{Character: c, LabelID: 3})
	createMailLabel(model.MailLabel{Character: c, LabelID: 7})
	createMailLabel(model.MailLabel{Character: c, LabelID: 13})
	// when
	ll, err := model.FetchMailLabels(c.ID, []int32{3, 13})
	if assert.NoError(t, err) {
		gotIDs := set.New[int32]()
		for _, l := range ll {
			gotIDs.Add(l.LabelID)
		}
		wantIDs := set.NewFromSlice([]int32{3, 13})
		assert.Equal(t, wantIDs, gotIDs)
	}
}
