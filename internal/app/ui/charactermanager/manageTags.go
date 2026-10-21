package charactermanager

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/go-set"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/filedialog"
	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type manageTags struct {
	widget.BaseWidget

	addCharactersButton *widget.Button
	characterList       *widget.List
	characters          []*app.EntityShort
	charactersLoad      atomic.Int64 // identifies the newest characters load
	emptyCharactersHint fyne.CanvasObject
	emptyTagsHint       fyne.CanvasObject
	manageCharacters    *xwidget.AppBar
	cw                  *manageCharacters
	selectedTag         *app.CharacterTag
	signalKey           string
	tagList             *widget.List
	tags                []*app.CharacterTag
	tagsLoad            atomic.Int64 // identifies the newest tags load
}

func newManageTags(cw *manageCharacters) *manageTags {
	a := &manageTags{
		cw:        cw,
		signalKey: cw.u.Signals().UniqueKey(),
	}
	a.ExtendBaseWidget(a)

	l1 := widget.NewLabel("No tags")
	l1.Importance = widget.LowImportance
	a.emptyTagsHint = container.NewCenter(l1)

	l2 := widget.NewLabel("No characters")
	l2.Importance = widget.LowImportance
	a.emptyCharactersHint = container.NewCenter(l2)

	a.addCharactersButton = a.makeAddCharacterButton()
	a.characterList = a.makeCharacterList()
	a.manageCharacters = a.makeManageCharacters()
	a.tagList = a.makeTagList()

	// Signals
	a.cw.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
		a.cw.u.Signals().TagsChanged.Emit(ctx, struct{}{})
	}, a.signalKey)
	return a
}

func (a *manageTags) stop() {
	a.cw.u.Signals().CharacterRemoved.RemoveListener(a.signalKey)
}

func (a *manageTags) CreateRenderer() fyne.WidgetRenderer {
	// p := theme.Padding()
	addTag := widget.NewButtonWithIcon("Create tag", theme.ContentAddIcon(), func() {
		a.modifyTag("Create Character Tag", "Create", func(ctx context.Context, name string) error {
			_, err := a.cw.u.Character().CreateTag(ctx, name)
			return err
		})
	})
	addTag.Importance = widget.HighImportance
	main := container.NewBorder(
		nil,
		container.NewVBox(addTag, xwidget.NewStandardSpacer()),
		nil,
		nil,
		a.tagList,
	)
	actions := xwidget.NewIconButtonWithMenu(theme.MoreHorizontalIcon(), fyne.NewMenu("",
		fyne.NewMenuItem("Export tags", a.exportTags),
		fyne.NewMenuItem("Import tags", a.importTags),
		fyne.NewMenuItem("Delete all tags", a.deleteTags),
	))
	ab := xwidget.NewAppBar("Tags", main, actions)
	ab.HideBackground = !a.cw.u.IsMobile()
	c := container.NewVSplit(
		container.NewStack(ab, a.emptyTagsHint),
		container.NewStack(a.manageCharacters, a.emptyCharactersHint),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *manageTags) deleteTags() {
	ui.ShowLoadingConfirm(
		a.cw.u,
		"Delete all tags?",
		"This will permanently remove all tags",
		"Delete",
		widget.DangerImportance,
		func(ctx context.Context) {
			a.deleteAllTags(ctx)
		},
		a.cw.w,
	)
}

func (a *manageTags) deleteAllTags(ctx context.Context) {
	err := a.cw.u.Character().DeleteAllTags(ctx)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		a.cw.reportError("Failed to delete tags", err)
		return
	}
	a.update(ctx)
	runAsync(func() {
		a.cw.u.Signals().TagsChanged.Emit(ctx, struct{}{})
	})
	a.cw.displaySnackbar("All tags deleted")
}

func (a *manageTags) exportTags() {
	filedialog.ShowSave(a.cw.u, filedialog.ShowFileSaveWindowParams{
		CompletionText:  "Tags exported",
		DisplaySnackbar: a.cw.displaySnackbar,
		Title:           "Export tags",
		WindowID:        "tags-export",
		WriteFunc: func(ctx context.Context, w io.Writer) error {
			return a.cw.u.Character().WriteTags(ctx, w, fyne.CurrentApp().Metadata().Version)
		},
		Window: a.cw.w,
	})
}

func (a *manageTags) importTags() {
	filedialog.ShowOpen(a.cw.u, filedialog.ShowFileOpenWindowParams{
		CompletionText:  "Tags imported",
		DisplaySnackbar: a.cw.displaySnackbar,
		Title:           "Import tags",
		WindowID:        "tags-import",
		ReadFunc:        a.importTagsFrom,
		Window:          a.cw.w,
	})
}

func (a *manageTags) importTagsFrom(ctx context.Context, r io.Reader) error {
	err := a.cw.u.Character().ReadAndReplaceTags(ctx, r, fyne.CurrentApp().Metadata().Version)
	if err != nil {
		return err
	}
	a.update(ctx)
	runAsync(func() {
		a.cw.u.Signals().TagsChanged.Emit(ctx, struct{}{})
	})
	return nil
}

func (a *manageTags) makeManageCharacters() *xwidget.AppBar {
	ab := xwidget.NewAppBar(
		"",
		container.NewBorder(
			nil,
			container.NewVBox(a.addCharactersButton, xwidget.NewStandardSpacer()),
			nil,
			nil,
			a.characterList,
		),
	)
	ab.HideBackground = !a.cw.u.IsMobile()
	ab.Hide()
	return ab
}

func (a *manageTags) makeAddCharacterButton() *widget.Button {
	w := widget.NewButtonWithIcon("Add characters to tag", theme.ContentAddIcon(), func() {
		if a.selectedTag == nil {
			return
		}
		tag := a.selectedTag
		runAsync(func() {
			_, others, err := a.cw.u.Character().ListCharactersForTag(context.Background(), tag.ID)
			if err != nil {
				a.cw.reportError("Failed to list characters", err)
				return
			}
			if len(others) == 0 {
				return
			}
			fyne.Do(func() {
				selected := make(map[int64]bool)
				list := widget.NewList(
					func() int {
						return len(others)
					},
					func() fyne.CanvasObject {
						check := widget.NewIcon(theme.CheckButtonIcon())
						character := ui.NewEveEntityListItem(a.cw.u.EVEImage().EveEntityLogoAsync)
						character.IsAvatar = true
						return container.NewBorder(
							nil,
							nil,
							check,
							nil,
							character,
						)
					},
					func(id widget.ListItemID, co fyne.CanvasObject) {
						if id >= len(others) {
							return
						}
						border := co.(*fyne.Container).Objects
						r := others[id]
						border[0].(*ui.EveEntityListItem).Set2(r.ID, r.Name, app.EveEntityCharacter)

						check := border[1].(*widget.Icon)
						if selected[r.ID] {
							check.SetResource(theme.CheckButtonCheckedIcon())
						} else {
							check.SetResource(theme.CheckButtonIcon())
						}
					},
				)
				list.HideSeparators = true
				list.OnSelected = func(id widget.ListItemID) {
					list.UnselectAll()
					if id >= len(others) {
						return
					}
					character := others[id]
					selected[character.ID] = !selected[character.ID]
					list.RefreshItem(id)
				}
				d := dialog.NewCustomConfirm(
					"Add characters to tag: "+tag.Name,
					"Add",
					"Cancel",
					list,
					func(confirmed bool) {
						if !confirmed {
							return
						}
						var characterIDs []int64
						for characterID, v := range selected {
							if v {
								characterIDs = append(characterIDs, characterID)
							}
						}
						runAsync(func() {
							a.addCharactersToTag(context.Background(), tag, characterIDs)
						})
					},
					a.cw.w,
				)
				xdesktop.DisableShortcutsForDialog(d, a.cw.w)
				d.Show()
				_, s := a.cw.w.Canvas().InteractiveArea()
				d.Resize(fyne.NewSize(s.Width*0.8, s.Height*0.8))
			})
		})
	})
	w.Importance = widget.HighImportance
	w.Disable()
	return w
}

func (a *manageTags) addCharactersToTag(ctx context.Context, tag *app.CharacterTag, characterIDs []int64) {
	var added int
	for _, characterID := range characterIDs {
		err := a.cw.u.Character().AddTagToCharacter(ctx, characterID, tag.ID)
		if err != nil {
			a.cw.reportError("Failed to add tag to character", err)
			break
		}
		added++
	}
	if added == 0 {
		return
	}
	fyne.Do(func() {
		a.setCharactersAsync(tag)
	})
	runAsync(func() {
		a.cw.u.Signals().TagsChanged.Emit(ctx, struct{}{})
	})
}

func (a *manageTags) makeTagList() *widget.List {
	tagList := widget.NewList(
		func() int {
			return len(a.tags)
		},
		func() fyne.CanvasObject {
			del := ttwidget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
			del.Importance = widget.DangerImportance
			del.SetToolTip("Delete tag")
			rename := ttwidget.NewButtonWithIcon("", theme.DocumentCreateIcon(), nil)
			rename.SetToolTip("Rename tag")
			name := widget.NewLabel("Template")
			return container.NewBorder(
				nil,
				nil,
				nil,
				container.NewHBox(rename, del),
				name,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.tags) {
				return
			}
			tag := a.tags[id]
			box := co.(*fyne.Container).Objects
			box[0].(*widget.Label).SetText(tag.Name)
			icons := box[1].(*fyne.Container).Objects
			icons[0].(*ttwidget.Button).OnTapped = func() {
				a.modifyTag("Rename tag: "+tag.Name, "Rename", func(ctx context.Context, name string) error {
					return a.cw.u.Character().RenameTag(ctx, tag.ID, name)
				})
			}
			icons[1].(*ttwidget.Button).OnTapped = func() {
				ui.ShowLoadingConfirm(
					a.cw.u,
					"Delete Tag?",
					"This will permanently delete tag \""+tag.Name+"\"",
					"Delete",
					widget.DangerImportance,
					func(ctx context.Context) {
						a.deleteTag(ctx, tag)
					}, a.cw.w,
				)
			}
		},
	)
	tagList.HideSeparators = true
	tagList.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.tags) {
			tagList.UnselectAll()
			return
		}
		tag := a.tags[id]
		a.setCharactersAsync(tag)
	}
	return tagList
}

func (a *manageTags) deleteTag(ctx context.Context, tag *app.CharacterTag) {
	err := a.cw.u.Character().DeleteTag(ctx, tag.ID)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		a.cw.reportError("Failed to delete tag", err)
		return
	}
	a.update(ctx)
	runAsync(func() {
		a.cw.u.Signals().TagsChanged.Emit(ctx, struct{}{})
	})
}

func (a *manageTags) makeCharacterList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			remove := ttwidget.NewButtonWithIcon("", theme.CancelIcon(), nil)
			remove.SetToolTip("Remove character from tag")
			character := ui.NewEveEntityListItem(a.cw.u.EVEImage().EveEntityLogoAsync)
			character.IsAvatar = true
			return container.NewBorder(
				nil,
				nil,
				nil,
				remove,
				character,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.characters) {
				return
			}
			r := a.characters[id]
			box := co.(*fyne.Container).Objects
			box[0].(*ui.EveEntityListItem).Set2(r.ID, r.Name, app.EveEntityCharacter)

			remove := box[1].(*ttwidget.Button)
			remove.OnTapped = func() {
				if a.selectedTag == nil {
					return
				}
				tag := a.selectedTag
				runAsync(func() {
					a.removeCharacterFromTag(context.Background(), tag, r.ID)
				})
			}
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(_ widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *manageTags) removeCharacterFromTag(ctx context.Context, tag *app.CharacterTag, characterID int64) {
	err := a.cw.u.Character().RemoveTagFromCharacter(ctx, characterID, tag.ID)
	if err != nil {
		a.cw.reportError("Failed to remove tag from character: "+tag.Name, err)
		return
	}
	fyne.Do(func() {
		a.setCharactersAsync(tag)
	})
	runAsync(func() {
		a.cw.u.Signals().TagsChanged.Emit(ctx, struct{}{})
	})
}

func (a *manageTags) setCharactersAsync(tag *app.CharacterTag) {
	switched := tag == nil || a.selectedTag == nil || a.selectedTag.ID != tag.ID
	a.selectedTag = tag
	load := a.charactersLoad.Add(1)
	if switched {
		// Don't show the previous tag's characters under the new tag's title.
		xslices.Clear(&a.characters)
		a.characterList.Refresh()
		a.addCharactersButton.Disable()
	}
	if tag == nil {
		a.manageCharacters.Hide()
		a.emptyCharactersHint.Show()
		return
	}
	a.manageCharacters.SetTitle("Tag: " + tag.Name)
	a.manageCharacters.Show()
	runAsync(func() {
		tagged, others, err := a.cw.u.Character().ListCharactersForTag(context.Background(), tag.ID)
		if err != nil {
			if load != a.charactersLoad.Load() {
				return // outdated
			}
			a.cw.reportError("Failed to list characters for "+tag.Name, err)
			return
		}
		fyne.Do(func() {
			if load != a.charactersLoad.Load() {
				return // outdated
			}
			a.characters = tagged
			a.characterList.Refresh()
			if len(others) > 0 {
				a.addCharactersButton.Enable()
			} else {
				a.addCharactersButton.Disable()
			}
			if len(tagged) > 0 {
				a.emptyCharactersHint.Hide()
			} else {
				a.emptyCharactersHint.Show()
			}
		})
	})
}

func (a *manageTags) modifyTag(title, confirm string, execute func(ctx context.Context, name string) error) {
	name := widget.NewEntry()
	name.Validator = newTagNameValidator(a.tags)
	items := []*widget.FormItem{
		widget.NewFormItem("Name", name),
	}
	d := dialog.NewForm(
		title, confirm, "Cancel", items, func(confirmed bool) {
			if !confirmed {
				return
			}
			a.submitTag(context.Background(), name.Text, execute)
		}, a.cw.w,
	)
	xdesktop.DisableShortcutsForDialog(d, a.cw.w)
	d.Show()
	d.Resize(fyne.NewSize(300, 200))
	a.cw.w.Canvas().Focus(name)
}

func (a *manageTags) submitTag(ctx context.Context, name string, execute func(ctx context.Context, name string) error) {
	if err := execute(ctx, name); err != nil {
		ui.NotifyErrorAndLog("Failed to modify tag", err, a.cw.displaySnackbar)
		return
	}
	a.update(ctx)
	runAsync(func() {
		a.cw.u.Signals().TagsChanged.Emit(ctx, struct{}{})
	})
}

func newTagNameValidator(tags []*app.CharacterTag) fyne.StringValidator {
	names := set.Of(xslices.Map(tags, func(x *app.CharacterTag) string {
		return strings.ToLower(x.Name)
	})...)
	return func(s string) error {
		if len(s) == 0 {
			return errors.New("can not be empty")
		}
		if names.Contains(strings.ToLower(s)) {
			return errors.New("tag with same name already exists")
		}
		return nil
	}
}

func (a *manageTags) selectTagByName(name string) {
	a.tagList.UnselectAll()
	for id, t := range a.tags {
		if t.Name == name {
			a.tagList.Select(id)
			break
		}
	}
}

func (a *manageTags) update(ctx context.Context) {
	load := a.tagsLoad.Add(1)
	tags, err := a.cw.u.Character().ListTagsByName(ctx)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		if load != a.tagsLoad.Load() {
			return // outdated
		}
		a.cw.reportError("Failed to list tags", err)
		return
	}
	fyne.Do(func() {
		if load != a.tagsLoad.Load() {
			return // outdated
		}
		a.tags = tags
		a.tagList.Refresh()
		a.tagList.UnselectAll()
		if len(tags) > 0 {
			a.emptyTagsHint.Hide()
			a.selectTagByName(tags[0].Name) // loads its characters via OnSelected
		} else {
			a.emptyTagsHint.Show()
			a.setCharactersAsync(nil)
		}
	})
}
