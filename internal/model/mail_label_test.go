package model_test

import (
	"example/evebuddy/internal/factory"
	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/model"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMailLabelSaveNew(t *testing.T) {
	// given
	model.TruncateTables()
	c := factory.CreateCharacter()
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
	l := factory.CreateMailLabel()
	// when
	l2, err := model.FetchMailLabel(l.Character.ID, l.LabelID)
	if assert.NoError(t, err) {
		assert.Equal(t, l.Name, l2.Name)
	}
}

func TestCanFetchAllMailLabelsForCharacter(t *testing.T) {
	// given
	model.TruncateTables()
	c1 := factory.CreateCharacter()
	factory.CreateMailLabel(model.MailLabel{Character: c1, LabelID: model.LabelAlliance})
	l1 := factory.CreateMailLabel(model.MailLabel{Character: c1, LabelID: 103})
	fmt.Println(l1)
	l2 := factory.CreateMailLabel(model.MailLabel{Character: c1, LabelID: 107})
	fmt.Println(l2)
	c2 := factory.CreateCharacter()
	factory.CreateMailLabel(model.MailLabel{Character: c2, LabelID: 113})
	// when
	ll, err := model.FetchCustomMailLabels(c1.ID)
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
	c := factory.CreateCharacter()
	// when
	ll, err := model.FetchCustomMailLabels(c.ID)
	if assert.NoError(t, err) {
		assert.Empty(t, ll)
	}
}

func TestFetchMailLabels(t *testing.T) {
	// given
	model.TruncateTables()
	c := factory.CreateCharacter()
	factory.CreateMailLabel(model.MailLabel{Character: c, LabelID: 3})
	factory.CreateMailLabel(model.MailLabel{Character: c, LabelID: 7})
	factory.CreateMailLabel(model.MailLabel{Character: c, LabelID: 13})
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
