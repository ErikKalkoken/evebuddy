package repository_test

import (
	"context"
	"example/evebuddy/internal/repository"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMailLabel(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		arg := repository.MailLabelParams{
			CharacterID: c.ID,
			Color:       "xyz",
			LabelID:     42,
			Name:        "Dummy",
			UnreadCount: 99,
		}
		// when
		_, err := r.UpdateOrCreateMailLabel(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetMailLabel(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Dummy", l.Name)
				assert.Equal(t, "xyz", l.Color)
				assert.Equal(t, 99, l.UnreadCount)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateMailLabel(repository.MailLabel{CharacterID: c.ID, LabelID: 42})
		arg := repository.MailLabelParams{
			CharacterID: c.ID,
			Color:       "xyz",
			LabelID:     42,
			Name:        "Dummy",
			UnreadCount: 99,
		}
		// when
		_, err := r.UpdateOrCreateMailLabel(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetMailLabel(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Dummy", l.Name)
				assert.Equal(t, "xyz", l.Color)
				assert.Equal(t, 99, l.UnreadCount)
			}
		}
	})
	t.Run("can get or create existing", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateMailLabel(repository.MailLabel{CharacterID: c.ID, LabelID: 42, Name: "Dummy"})
		arg := repository.MailLabelParams{
			CharacterID: c.ID,
			Color:       "xyz",
			LabelID:     42,
			Name:        "Johnny",
			UnreadCount: 99,
		}
		// when
		_, err := r.GetOrCreateMailLabel(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetMailLabel(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Dummy", l.Name)
			}
		}
	})
	t.Run("can get or create when not existing", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		arg := repository.MailLabelParams{
			CharacterID: c.ID,
			Color:       "xyz",
			LabelID:     42,
			Name:        "Johnny",
			UnreadCount: 99,
		}
		// when
		_, err := r.GetOrCreateMailLabel(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetMailLabel(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Johnny", l.Name)
			}
		}
	})
	t.Run("can return all mail labels for a character ordered by name", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateMailLabel(repository.MailLabel{CharacterID: c.ID, Name: "bravo"})
		l2 := factory.CreateMailLabel(repository.MailLabel{CharacterID: c.ID, Name: "alpha"})
		factory.CreateMailLabel()
		// when
		got, err := r.ListMailLabelsOrdered(ctx, c.ID)
		if assert.NoError(t, err) {
			want := []repository.MailLabel{l2, l1}
			assert.Equal(t, want, got)
		}
	})
}

// TODO: Reimplement tests

// func TestCanFetchAllMailLabelsForCharacter(t *testing.T) {
// 	// given
// 	repository.TruncateTables()
// 	c1 := factory.CreateCharacter()
// 	factory.CreateMailLabel(repository.MailLabel{Character: c1, LabelID: repository.LabelAlliance})
// 	l1 := factory.CreateMailLabel(repository.MailLabel{Character: c1, LabelID: 103})
// 	fmt.Println(l1)
// 	l2 := factory.CreateMailLabel(repository.MailLabel{Character: c1, LabelID: 107})
// 	fmt.Println(l2)
// 	c2 := factory.CreateCharacter()
// 	factory.CreateMailLabel(repository.MailLabel{Character: c2, LabelID: 113})
// 	// when
// 	ll, err := repository.ListMailLabels(c1.ID)
// 	if assert.NoError(t, err) {
// 		gotIDs := set.New[int32]()
// 		for _, l := range ll {
// 			gotIDs.Add(l.LabelID)
// 		}
// 		wantIDs := set.NewFromSlice([]int32{103, 107})
// 		assert.Equal(t, wantIDs, gotIDs)
// 	}
// }

// func TesFetchAllMailLabelsReturnsEmptySliceWhenNoRows(t *testing.T) {
// 	// given
// 	repository.TruncateTables()
// 	c := factory.CreateCharacter()
// 	// when
// 	ll, err := repository.ListMailLabels(c.ID)
// 	if assert.NoError(t, err) {
// 		assert.Empty(t, ll)
// 	}
// }

// func TestFetchMailLabels(t *testing.T) {
// 	// given
// 	repository.TruncateTables()
// 	c := factory.CreateCharacter()
// 	factory.CreateMailLabel(repository.MailLabel{Character: c, LabelID: 3})
// 	factory.CreateMailLabel(repository.MailLabel{Character: c, LabelID: 7})
// 	factory.CreateMailLabel(repository.MailLabel{Character: c, LabelID: 13})
// 	// when
// 	ll, err := repository.ListMailLabelsByIDs(c.ID, []int32{3, 13})
// 	if assert.NoError(t, err) {
// 		gotIDs := set.New[int32]()
// 		for _, l := range ll {
// 			gotIDs.Add(l.LabelID)
// 		}
// 		wantIDs := set.NewFromSlice([]int32{3, 13})
// 		assert.Equal(t, wantIDs, gotIDs)
// 	}
// }
