package charactermanager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestManageTags_Update(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("selects first tag and shows its characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		factory.CreateCharacterTag("Bravo")
		alpha := factory.CreateCharacterTag("Alpha")
		c1 := factory.CreateCharacter()
		factory.AddCharacterToTag(alpha, c1)
		factory.CreateCharacter()
		a := cw.characterTags
		// when
		a.update(context.Background())
		// then
		var names []string
		for _, x := range a.tags {
			names = append(names, x.Name)
		}
		assert.Equal(t, []string{"Alpha", "Bravo"}, names)
		require.NotNil(t, a.selectedTag)
		assert.Equal(t, alpha.ID, a.selectedTag.ID)
		if assert.Len(t, a.characters, 1) {
			assert.Equal(t, c1.ID, a.characters[0].ID)
		}
		assert.False(t, a.addCharactersButton.Disabled())
		assert.False(t, a.emptyTagsHint.Visible())
		assert.False(t, a.emptyCharactersHint.Visible())
		assert.True(t, a.manageCharacters.Visible())
		assert.Equal(t, "Tag: Alpha", a.manageCharacters.Title())
	})
	t.Run("shows hints when there are no tags", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		a := cw.characterTags
		// when
		a.update(context.Background())
		// then
		assert.Empty(t, a.tags)
		assert.Nil(t, a.selectedTag)
		assert.True(t, a.emptyTagsHint.Visible())
		assert.True(t, a.emptyCharactersHint.Visible())
		assert.False(t, a.manageCharacters.Visible())
	})
	t.Run("loads characters of the selected tag once", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		factory.CreateCharacterTag("Alpha")
		a := cw.characterTags
		var pending []func()
		orig := runAsync
		runAsync = func(f func()) { pending = append(pending, f) }
		t.Cleanup(func() { runAsync = orig })
		// when
		a.update(context.Background())
		// then
		assert.Len(t, pending, 1)
	})
	t.Run("reports error and keeps tags when loading fails", func(t *testing.T) {
		// given
		db2, st2, factory2 := testutil.NewDBInMemory()
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st2})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		var messages []string
		cw.displaySnackbar = func(s string) {
			messages = append(messages, s)
		}
		tag := factory2.CreateCharacterTag("Alpha")
		a := cw.characterTags
		a.update(context.Background())
		db2.Close()
		// when
		a.update(context.Background())
		// then
		if assert.Len(t, a.tags, 1) {
			assert.Equal(t, tag.ID, a.tags[0].ID)
		}
		require.NotNil(t, a.selectedTag)
		assert.Equal(t, tag.ID, a.selectedTag.ID)
		if assert.Len(t, messages, 1) {
			assert.Contains(t, messages[0], "ERROR: Failed to list tags")
		}
	})
	t.Run("reloads tags and emits TagsChanged when a character was removed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		factory.CreateCharacterTag("Alpha")
		var tagsChanged int
		u.Signals().TagsChanged.AddListener(func(context.Context, struct{}) {
			tagsChanged++
		})
		// when
		u.Signals().CharacterRemoved.Emit(context.Background(), &app.EntityShort{ID: 42})
		// then
		if assert.Len(t, cw.characterTags.tags, 1) {
			assert.Equal(t, "Alpha", cw.characterTags.tags[0].Name)
		}
		assert.Equal(t, 1, tagsChanged)
	})
}

func TestManageTags_SetCharacters(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("selecting a tag shows its characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		factory.CreateCharacterTag("Alpha")
		bravo := factory.CreateCharacterTag("Bravo")
		c := factory.CreateCharacter()
		factory.AddCharacterToTag(bravo, c)
		a := cw.characterTags
		a.update(context.Background())
		// when
		a.tagList.OnSelected(1)
		// then
		require.NotNil(t, a.selectedTag)
		assert.Equal(t, bravo.ID, a.selectedTag.ID)
		if assert.Len(t, a.characters, 1) {
			assert.Equal(t, c.ID, a.characters[0].ID)
		}
		assert.Equal(t, "Tag: Bravo", a.manageCharacters.Title())
	})
	t.Run("disables add button when all characters have the tag", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		tag := factory.CreateCharacterTag("Alpha")
		c := factory.CreateCharacter()
		factory.AddCharacterToTag(tag, c)
		a := cw.characterTags
		// when
		a.setCharactersAsync(tag)
		// then
		assert.True(t, a.addCharactersButton.Disabled())
		assert.False(t, a.emptyCharactersHint.Visible())
	})
	t.Run("shows hint when no character has the tag", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		tag := factory.CreateCharacterTag("Alpha")
		factory.CreateCharacter()
		a := cw.characterTags
		// when
		a.setCharactersAsync(tag)
		// then
		assert.Empty(t, a.characters)
		assert.False(t, a.addCharactersButton.Disabled())
		assert.True(t, a.emptyCharactersHint.Visible())
	})
	t.Run("ignores outdated load when switching tags", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		alpha := factory.CreateCharacterTag("Alpha")
		bravo := factory.CreateCharacterTag("Bravo")
		c1 := factory.CreateCharacter()
		factory.AddCharacterToTag(alpha, c1)
		c2 := factory.CreateCharacter()
		factory.AddCharacterToTag(bravo, c2)
		a := cw.characterTags
		var pending []func()
		orig := runAsync
		runAsync = func(f func()) { pending = append(pending, f) }
		t.Cleanup(func() { runAsync = orig })
		a.setCharactersAsync(alpha)
		a.setCharactersAsync(bravo)
		runAsync = orig
		// when
		pending[1]() // Bravo finishes first
		pending[0]()
		// then
		if assert.Len(t, a.characters, 1) {
			assert.Equal(t, c2.ID, a.characters[0].ID)
		}
	})
	t.Run("ignores pending load after clearing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		tag := factory.CreateCharacterTag("Alpha")
		c := factory.CreateCharacter()
		factory.AddCharacterToTag(tag, c)
		a := cw.characterTags
		var pending []func()
		orig := runAsync
		runAsync = func(f func()) { pending = append(pending, f) }
		t.Cleanup(func() { runAsync = orig })
		a.setCharactersAsync(tag)
		runAsync = orig
		a.setCharactersAsync(nil)
		// when
		pending[0]()
		// then
		assert.Empty(t, a.characters)
		assert.True(t, a.emptyCharactersHint.Visible())
	})
	t.Run("clears characters when no tag is given", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		tag := factory.CreateCharacterTag("Alpha")
		c := factory.CreateCharacter()
		factory.AddCharacterToTag(tag, c)
		factory.CreateCharacter() // enables the add button
		a := cw.characterTags
		a.setCharactersAsync(tag)
		// when
		a.setCharactersAsync(nil)
		// then
		assert.Nil(t, a.selectedTag)
		assert.Empty(t, a.characters)
		assert.True(t, a.addCharactersButton.Disabled())
		assert.False(t, a.manageCharacters.Visible())
		assert.True(t, a.emptyCharactersHint.Visible())
	})
}

func TestNewTagNameValidator(t *testing.T) {
	tags := []*app.CharacterTag{{ID: 1, Name: "Alpha"}}
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"accepts new name", "Bravo", false},
		{"rejects empty name", "", true},
		{"rejects existing name", "Alpha", true},
		{"rejects existing name in other case", "aLPHA", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			v := newTagNameValidator(tags)
			// when
			err := v(tc.input)
			// then
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManageTags_SubmitTag(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("reloads tags and emits TagsChanged on success", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		tagsChanged := make(chan struct{}, 1)
		u.Signals().TagsChanged.AddListener(func(context.Context, struct{}) {
			tagsChanged <- struct{}{}
		})
		a := cw.characterTags
		// when
		a.submitTag(context.Background(), "Alpha", func(ctx context.Context, name string) error {
			_, err := u.Character().CreateTag(ctx, name)
			return err
		})
		// then
		if assert.Len(t, a.tags, 1) {
			assert.Equal(t, "Alpha", a.tags[0].Name)
		}
		select {
		case <-tagsChanged:
		case <-time.After(5 * time.Second):
			t.Fatal("TagsChanged not emitted")
		}
	})
	t.Run("shows error and keeps tags on failure", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		var messages []string
		cw.displaySnackbar = func(s string) {
			messages = append(messages, s)
		}
		factory.CreateCharacterTag("Alpha")
		a := cw.characterTags
		a.update(context.Background())
		// when
		a.submitTag(context.Background(), "Bravo", func(context.Context, string) error {
			return errors.New("dummy")
		})
		// then
		assert.Equal(t, []string{"Failed to modify tag"}, messages)
		if assert.Len(t, a.tags, 1) {
			assert.Equal(t, "Alpha", a.tags[0].Name)
		}
	})
}

func TestManageTags_DeleteTag(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("selects first remaining tag", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		alpha := factory.CreateCharacterTag("Alpha")
		bravo := factory.CreateCharacterTag("Bravo")
		a := cw.characterTags
		a.update(context.Background())
		// when
		a.deleteTag(context.Background(), alpha)
		// then
		if assert.Len(t, a.tags, 1) {
			assert.Equal(t, bravo.ID, a.tags[0].ID)
		}
		require.NotNil(t, a.selectedTag)
		assert.Equal(t, bravo.ID, a.selectedTag.ID)
	})
	t.Run("clears selection when last tag was deleted", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		tag := factory.CreateCharacterTag("Alpha")
		c := factory.CreateCharacter()
		factory.AddCharacterToTag(tag, c)
		a := cw.characterTags
		a.update(context.Background())
		// when
		a.deleteTag(context.Background(), tag)
		// then
		assert.Empty(t, a.tags)
		assert.Nil(t, a.selectedTag)
		assert.Empty(t, a.characters)
		assert.True(t, a.addCharactersButton.Disabled())
		assert.False(t, a.manageCharacters.Visible())
		assert.True(t, a.emptyTagsHint.Visible())
	})
}

func TestManageTags_DeleteAllTags(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("deletes all tags and reports it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		var messages []string
		cw.displaySnackbar = func(s string) {
			messages = append(messages, s)
		}
		tagsChanged := make(chan struct{}, 1)
		u.Signals().TagsChanged.AddListener(func(context.Context, struct{}) {
			tagsChanged <- struct{}{}
		})
		factory.CreateCharacterTag("Alpha")
		factory.CreateCharacterTag("Bravo")
		a := cw.characterTags
		a.update(context.Background())
		// when
		a.deleteAllTags(context.Background())
		// then
		assert.Empty(t, a.tags)
		assert.Equal(t, []string{"All tags deleted"}, messages)
		select {
		case <-tagsChanged:
		case <-time.After(5 * time.Second):
			t.Fatal("TagsChanged not emitted")
		}
	})
}

func TestManageTags_AddCharactersToTag(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("adds characters and shows them", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		tagsChanged := make(chan struct{}, 1)
		u.Signals().TagsChanged.AddListener(func(context.Context, struct{}) {
			tagsChanged <- struct{}{}
		})
		tag := factory.CreateCharacterTag("Alpha")
		c1 := factory.CreateCharacter()
		c2 := factory.CreateCharacter()
		a := cw.characterTags
		a.update(context.Background())
		// when
		a.addCharactersToTag(context.Background(), tag, []int64{c1.ID, c2.ID})
		// then
		var got []int64
		for _, c := range a.characters {
			got = append(got, c.ID)
		}
		assert.ElementsMatch(t, []int64{c1.ID, c2.ID}, got)
		assert.True(t, a.addCharactersButton.Disabled())
		select {
		case <-tagsChanged:
		case <-time.After(5 * time.Second):
			t.Fatal("TagsChanged not emitted")
		}
	})
}

func TestManageTags_RemoveCharacterFromTag(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("removes character and updates the list", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		tagsChanged := make(chan struct{}, 1)
		u.Signals().TagsChanged.AddListener(func(context.Context, struct{}) {
			tagsChanged <- struct{}{}
		})
		tag := factory.CreateCharacterTag("Alpha")
		c := factory.CreateCharacter()
		factory.AddCharacterToTag(tag, c)
		a := cw.characterTags
		a.update(context.Background())
		// when
		a.removeCharacterFromTag(context.Background(), tag, c.ID)
		// then
		assert.Empty(t, a.characters)
		assert.False(t, a.addCharactersButton.Disabled())
		assert.True(t, a.emptyCharactersHint.Visible())
		select {
		case <-tagsChanged:
		case <-time.After(5 * time.Second):
			t.Fatal("TagsChanged not emitted")
		}
	})
}

// appWithVersion is a Fyne app that reports a version, which importing tags requires.
type appWithVersion struct {
	fyne.App
}

func (a appWithVersion) Metadata() fyne.AppMetadata {
	return fyne.AppMetadata{Version: "1.2.3"}
}

func TestManageTags_ImportTagsFrom(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("replaces tags and shows them", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		fyneApp := appWithVersion{test.NewTempApp(t)}
		fyne.SetCurrentApp(fyneApp)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: fyneApp, Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		tagsChanged := make(chan struct{}, 1)
		u.Signals().TagsChanged.AddListener(func(context.Context, struct{}) {
			tagsChanged <- struct{}{}
		})
		factory.CreateCharacterTag("Alpha")
		c := factory.CreateCharacter()
		a := cw.characterTags
		a.update(context.Background())
		data := fmt.Sprintf(`{"Tags":{"Bravo":[%d]},"Version":"1.2.3"}`, c.ID)
		// when
		err := a.importTagsFrom(context.Background(), strings.NewReader(data))
		// then
		require.NoError(t, err)
		if assert.Len(t, a.tags, 1) {
			assert.Equal(t, "Bravo", a.tags[0].Name)
		}
		if assert.Len(t, a.characters, 1) {
			assert.Equal(t, c.ID, a.characters[0].ID)
		}
		select {
		case <-tagsChanged:
		case <-time.After(5 * time.Second):
			t.Fatal("TagsChanged not emitted")
		}
	})
	t.Run("returns error and keeps tags when data is invalid", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		factory.CreateCharacterTag("Alpha")
		a := cw.characterTags
		a.update(context.Background())
		// when
		err := a.importTagsFrom(context.Background(), strings.NewReader("invalid"))
		// then
		assert.Error(t, err)
		if assert.Len(t, a.tags, 1) {
			assert.Equal(t, "Alpha", a.tags[0].Name)
		}
	})
}
