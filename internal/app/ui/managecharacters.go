package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	kmodal "github.com/ErikKalkoken/fyne-kx/modal"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type accountCharacter struct {
	id           int32
	name         string
	missingToken bool
}

type manageCharactersWindow struct {
	sb          *iwidget.Snackbar
	tagsChanged bool
	u           *baseUI
	w           fyne.Window
}

func (a *manageCharactersWindow) reportError(text string, err error) {
	slog.Error(text, "error", err)
	a.sb.Show(fmt.Sprintf("ERROR: %s: %s", text, err))
}

type manageCharacters struct {
	widget.BaseWidget

	ab         *iwidget.AppBar
	characters []accountCharacter
	list       *widget.List
	mcw        *manageCharactersWindow
}

func showManageCharactersWindow(u *baseUI) {
	w, created, onClosed := u.getOrCreateWindowWithOnClosed("manage-characters", "Manage Characters")
	if !created {
		w.Show()
		return
	}
	mcw := &manageCharactersWindow{
		sb: iwidget.NewSnackbar(w),
		u:  u,
		w:  w,
	}
	characters := newManageCharacters(mcw)
	characters.update()
	c := container.NewAppTabs(
		container.NewTabItem("Characters", characters),
		container.NewTabItem("Tags", newCharacterTags(mcw)),
	)
	c.SetTabLocation(container.TabLocationLeading)
	w.SetContent(fynetooltip.AddWindowToolTipLayer(c, w.Canvas()))
	w.Resize(fyne.Size{Width: 700, Height: 500})
	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		if mcw.tagsChanged {
			u.updateHome()
		}
		mcw.sb.Stop()
	})
	w.SetCloseIntercept(func() {
		w.Close()
		fynetooltip.DestroyWindowToolTipLayer(w.Canvas())
	})
	w.Show()
}

func newManageCharacters(mcw *manageCharactersWindow) *manageCharacters {
	a := &manageCharacters{
		characters: make([]accountCharacter, 0),
		mcw:        mcw,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeCharacterList()
	add := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		a.ShowAddCharacterDialog()
	})
	add.Importance = widget.HighImportance
	if a.mcw.u.IsOffline() {
		add.Disable()
	}
	a.ab = iwidget.NewAppBar("Characters", container.NewBorder(
		nil,
		container.NewVBox(add, newStandardSpacer()),
		nil,
		nil,
		a.list,
	))
	return a
}

func (a *manageCharacters) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.ab)
}

func (a *manageCharacters) makeCharacterList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			portrait := iwidget.NewImageFromResource(
				icons.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			name := widget.NewLabel("Template")
			delete := ttwidget.NewButtonWithIcon("", theme.DeleteIcon(), func() {})
			delete.Importance = widget.DangerImportance
			delete.SetToolTip("Delete character")
			issue := widget.NewLabel("Scope issue - please re-add!")
			issue.Importance = widget.WarningImportance
			issue.Hide()
			row := container.NewHBox(portrait, name, issue, layout.NewSpacer(), delete)
			return row
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.characters) {
				return
			}
			c := a.characters[id]
			row := co.(*fyne.Container).Objects

			portrait := row[0].(*canvas.Image)
			go a.mcw.u.updateCharacterAvatar(c.id, func(r fyne.Resource) {
				fyne.Do(func() {
					portrait.Resource = r
					portrait.Refresh()
				})
			})

			name := row[1].(*widget.Label)
			name.SetText(c.name)

			issue := row[2].(*widget.Label)
			if c.missingToken {
				issue.Show()
			} else {
				issue.Hide()
			}

			delete := row[4].(*ttwidget.Button)
			delete.OnTapped = func() {
				a.showDeleteDialog(c)
			}
		})

	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.characters) {
			return
		}
		c := a.characters[id]
		if err := a.mcw.u.loadCharacter(c.id); err != nil {
			slog.Error("load current character", "char", c, "err", err)
			return
		}
		a.mcw.u.updateStatus()
		a.mcw.w.Close()
	}
	return l
}

func (a *manageCharacters) showDeleteDialog(c accountCharacter) {
	a.mcw.u.ShowConfirmDialog(
		"Delete Character",
		fmt.Sprintf("Are you sure you want to delete %s with all it's locally stored data?", c.name),
		"Delete",
		func(confirmed bool) {
			if confirmed {
				m := kmodal.NewProgressInfinite(
					"Deleting character",
					fmt.Sprintf("Deleting %s...", c.name),
					func() error {
						err := a.mcw.u.cs.DeleteCharacter(context.TODO(), c.id)
						if err != nil {
							return err
						}
						a.update()
						return nil
					},
					a.mcw.w,
				)
				m.OnSuccess = func() {
					a.mcw.sb.Show(fmt.Sprintf("Character %s deleted", c.name))
					go func() {
						a.update()
						if a.mcw.u.currentCharacterID() == c.id {
							a.mcw.u.setAnyCharacter()
						}
						a.mcw.u.updateHome()
						a.mcw.u.updateStatus()
					}()
				}
				m.OnError = func(err error) {
					a.mcw.reportError(fmt.Sprintf("ERROR: Failed to delete character %s", c.name), err)
				}
				m.Start()
			}
		},
		a.mcw.w,
	)
}

func (a *manageCharacters) update() {
	characters := xslices.Map(a.mcw.u.scs.ListCharacters(), func(c *app.EntityShort[int32]) accountCharacter {
		return accountCharacter{id: c.ID, name: c.Name}
	})
	// hasToken, err := a.mcw.u.cs.HasTokenWithScopes(context.Background(), c.ID)
	// if err != nil {
	// 	slog.Error("Tried to check if character has token", "err", err)
	// 	hasToken = true // do not report error when state is unclear
	// }
	fyne.Do(func() {
		a.characters = characters
		a.list.Refresh()
		a.ab.SetTitle(fmt.Sprintf("Characters (%d)", len(characters)))
	})
}

func (a *manageCharacters) ShowAddCharacterDialog() {
	cancelCTX, cancel := context.WithCancel(context.Background())
	s := "Please follow instructions in your browser to add a new character."
	infoText := binding.BindString(&s)
	content := widget.NewLabelWithData(infoText)
	d1 := dialog.NewCustom(
		"Add Character",
		"Cancel",
		content,
		a.mcw.w,
	)
	a.mcw.u.ModifyShortcutsForDialog(d1, a.mcw.w)
	d1.SetOnClosed(cancel)
	go func() {
		characterID, err := func() (int32, error) {
			characterID, err := a.mcw.u.cs.UpdateOrCreateCharacterFromSSO(cancelCTX, infoText)
			if err != nil {
				return 0, err
			}
			a.update()
			return characterID, nil
		}()
		fyne.Do(func() {
			d1.Hide()
			if err != nil && !errors.Is(err, app.ErrAborted) {
				s := "Failed to add a new character"
				slog.Error(s, "error", err)
				a.mcw.u.showErrorDialog(s, err, a.mcw.w)
				return
			}
			go func() {
				if !a.mcw.u.hasCharacter() {
					a.mcw.u.loadCharacter(characterID)
				}
				a.mcw.u.updateStatus()
				a.mcw.u.updateHome()
				if a.mcw.u.isUpdateDisabled { // FIXME: temporary for testing. should be removed again.
					return
				}
				go a.mcw.u.updateCharacterAndRefreshIfNeeded(context.Background(), characterID, true)
			}()
		})
	}()
	fyne.Do(func() {
		d1.Show()
	})
}

type characterTags struct {
	widget.BaseWidget

	addCharactersButton *widget.Button
	characterList       *widget.List
	characters          []*app.EntityShort[int32]
	emptyCharactersHint fyne.CanvasObject
	emptyTagsHint       fyne.CanvasObject
	manageCharacters    *iwidget.AppBar
	selectedTag         *app.CharacterTag
	tagList             *widget.List
	tags                []*app.CharacterTag
	mcw                 *manageCharactersWindow
}

func newCharacterTags(mcw *manageCharactersWindow) *characterTags {
	a := &characterTags{
		characters: make([]*app.EntityShort[int32], 0),
		mcw:        mcw,
		tags:       make([]*app.CharacterTag, 0),
	}
	a.ExtendBaseWidget(a)

	l := widget.NewLabel("Click + to add a character to this tag")
	l.Importance = widget.LowImportance
	a.emptyCharactersHint = container.NewCenter(l)

	l = widget.NewLabel("Click + to add a tag")
	l.Importance = widget.LowImportance
	a.emptyTagsHint = container.NewCenter(l)

	a.addCharactersButton = a.makeAddCharacterButton()
	a.characterList = a.makeCharacterList()
	a.manageCharacters = a.makeManageCharacters()
	a.tagList = a.makeTagList()
	a.updateTags()
	if len(a.tags) > 0 {
		a.tagList.Select(0)
	}
	return a
}

func (a *characterTags) CreateRenderer() fyne.WidgetRenderer {
	// p := theme.Padding()
	addTag := widget.NewButtonWithIcon("Create tag",
		theme.ContentAddIcon(), func() {
			a.modifyTag("Create Character Tag", "Create", func(name string) error {
				_, err := a.mcw.u.cs.CreateTag(context.Background(), name)
				return err
			})
		},
	)
	addTag.Importance = widget.HighImportance
	manageTags := iwidget.NewAppBar(
		"Tags",
		container.NewBorder(nil, container.NewVBox(addTag, newStandardSpacer()), nil, nil, a.tagList),
	)
	c := container.NewVSplit(
		container.NewStack(manageTags, a.emptyTagsHint),
		container.NewStack(a.manageCharacters, a.emptyCharactersHint),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *characterTags) makeManageCharacters() *iwidget.AppBar {
	ab := iwidget.NewAppBar(
		"",
		container.NewBorder(
			nil,
			container.NewVBox(a.addCharactersButton, newStandardSpacer()),
			nil,
			nil,
			a.characterList,
		),
	)
	ab.Hide()
	return ab
}

func (a *characterTags) makeAddCharacterButton() *widget.Button {
	w := widget.NewButtonWithIcon("Add characters to tag", theme.ContentAddIcon(), func() {
		if a.selectedTag == nil {
			return
		}
		_, others, err := a.mcw.u.cs.ListCharactersForTag(context.Background(), a.selectedTag.ID)
		if err != nil {
			a.mcw.reportError("Failed to list characters", err)
			a.characters = make([]*app.EntityShort[int32], 0)
			return
		}
		if len(others) == 0 {
			return
		}
		selected := make(map[int32]bool)
		list := widget.NewList(
			func() int {
				return len(others)
			},
			func() fyne.CanvasObject {
				check := widget.NewIcon(theme.CheckButtonIcon())
				portrait := iwidget.NewImageFromResource(
					icons.Characterplaceholder64Jpeg,
					fyne.NewSquareSize(app.IconUnitSize),
				)
				return container.NewBorder(
					nil,
					nil,
					container.NewHBox(check, portrait),
					nil,
					widget.NewLabel("Template"),
				)
			},
			func(id widget.ListItemID, co fyne.CanvasObject) {
				if id >= len(others) {
					return
				}
				box := co.(*fyne.Container).Objects
				character := others[id]
				box[0].(*widget.Label).SetText(character.Name)
				icons := box[1].(*fyne.Container).Objects

				portrait := icons[1].(*canvas.Image)
				go a.mcw.u.updateCharacterAvatar(character.ID, func(r fyne.Resource) {
					fyne.Do(func() {
						portrait.Resource = r
						portrait.Refresh()
					})
				})

				check := icons[0].(*widget.Icon)
				if selected[character.ID] {
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
			"Add characters to tag: "+a.selectedTag.Name,
			"Add",
			"Cancel",
			list,
			func(confirmed bool) {
				if !confirmed {
					return
				}
				for characterID, v := range selected {
					if !v {
						return
					}
					err := a.mcw.u.cs.AddTagToCharacter(
						context.Background(),
						characterID,
						a.selectedTag.ID,
					)
					if err != nil {
						a.mcw.reportError("Failed to add tag to character", err)
						return
					}
				}
				a.updateCharacters(a.selectedTag)
				a.mcw.tagsChanged = true
			},
			a.mcw.w,
		)
		a.mcw.u.ModifyShortcutsForDialog(d, a.mcw.w)
		d.Show()
		_, s := a.mcw.w.Canvas().InteractiveArea()
		d.Resize(fyne.NewSize(s.Width*0.8, s.Height*0.8))
	})
	w.Importance = widget.HighImportance
	w.Disable()
	return w
}

func (a *characterTags) makeTagList() *widget.List {
	tagList := widget.NewList(
		func() int {
			return len(a.tags)
		},
		func() fyne.CanvasObject {
			delete := ttwidget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
			delete.Importance = widget.DangerImportance
			delete.SetToolTip("Delete tag")
			rename := ttwidget.NewButtonWithIcon("", theme.DocumentCreateIcon(), nil)
			rename.SetToolTip("Rename tag")
			name := widget.NewLabel("Template")
			return container.NewBorder(
				nil,
				nil,
				nil,
				container.NewHBox(rename, delete),
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
				a.modifyTag("Rename tag: "+tag.Name, "Rename", func(name string) error {
					return a.mcw.u.cs.RenameTag(context.Background(), tag.ID, name)
				})
			}
			icons[1].(*ttwidget.Button).OnTapped = func() {
				s := "Are you sure you want to delete tag " + tag.Name + "?"
				a.mcw.u.ShowConfirmDialog(
					"Delete Tag", s, "Delete", func(confirmed bool) {
						if !confirmed {
							return
						}
						err := a.mcw.u.cs.DeleteTag(context.Background(), tag.ID)
						if err != nil {
							a.mcw.u.showErrorDialog("Failed to delete tag", err, a.mcw.w)
							return
						}
						a.mcw.tagsChanged = true
						a.updateTags()
						if len(a.tags) > 0 {
							a.tagList.Select(0)
							return
						}
						a.tagList.UnselectAll()
						a.selectedTag = nil
						a.characters = make([]*app.EntityShort[int32], 0)
						a.addCharactersButton.Disable()
						a.characterList.Refresh()
						a.addCharactersButton.Disable()
						a.manageCharacters.Hide()
					}, a.mcw.w,
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
		a.updateCharacters(tag)
	}
	return tagList
}

func (a *characterTags) makeCharacterList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			remove := ttwidget.NewButtonWithIcon("", theme.CancelIcon(), nil)
			remove.SetToolTip("Remove character from tag")
			portrait := iwidget.NewImageFromResource(
				icons.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			name := widget.NewLabel("Template")
			return container.NewBorder(
				nil,
				nil,
				portrait,
				remove,
				name,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.characters) {
				return
			}
			character := a.characters[id]
			box := co.(*fyne.Container).Objects
			box[0].(*widget.Label).SetText(character.Name)

			portrait := box[1].(*canvas.Image)
			go a.mcw.u.updateCharacterAvatar(character.ID, func(r fyne.Resource) {
				fyne.Do(func() {
					portrait.Resource = r
					portrait.Refresh()
				})
			})

			remove := box[2].(*ttwidget.Button)
			remove.OnTapped = func() {
				if a.selectedTag == nil {
					return
				}
				err := a.mcw.u.cs.RemoveTagFromCharacter(
					context.Background(),
					character.ID,
					a.selectedTag.ID,
				)
				if err != nil {
					a.mcw.reportError("Failed to remove tag from character: "+a.selectedTag.Name, err)
					return
				}
				a.updateCharacters(a.selectedTag)
				a.mcw.tagsChanged = true
			}
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *characterTags) updateCharacters(tag *app.CharacterTag) {
	if tag == nil {
		return
	}
	a.selectedTag = tag
	a.manageCharacters.SetTitle("Tag: " + tag.Name)
	a.manageCharacters.Show()
	tagged, others, err := a.mcw.u.cs.ListCharactersForTag(context.Background(), tag.ID)
	if err != nil {
		a.mcw.reportError("Failed to list characters for "+tag.Name, err)
		a.characters = make([]*app.EntityShort[int32], 0)
		return
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
}

func (a *characterTags) modifyTag(title, confirm string, execute func(name string) error) {
	names := set.Of(xslices.Map(a.tags, func(x *app.CharacterTag) string {
		return strings.ToLower(x.Name)
	})...)
	name := widget.NewEntry()
	name.Validator = func(s string) error {
		if len(s) == 0 {
			return errors.New("can not be empty")
		}
		if names.Contains(strings.ToLower(s)) {
			return errors.New("tag with same name already exists")
		}
		return nil
	}
	items := []*widget.FormItem{
		widget.NewFormItem("Name", name),
	}
	d := dialog.NewForm(
		title, confirm, "Cancel", items, func(confirmed bool) {
			if !confirmed {
				return
			}
			if err := execute(name.Text); err != nil {
				a.mcw.u.showErrorDialog("Failed to modify tag", err, a.mcw.w)
				return
			}
			a.updateTags()
			a.mcw.tagsChanged = true
			a.selectTagByName(name.Text)
		}, a.mcw.w,
	)
	a.mcw.u.ModifyShortcutsForDialog(d, a.mcw.w)
	d.Show()
	d.Resize(fyne.NewSize(300, 200))
	a.mcw.w.Canvas().Focus(name)
}

func (a *characterTags) selectTagByName(name string) {
	a.tagList.UnselectAll()
	for id, t := range a.tags {
		if t.Name == name {
			a.tagList.Select(id)
			break
		}
	}
}

func (a *characterTags) updateTags() {
	tags, err := a.mcw.u.cs.ListTagsByName(context.Background())
	if err != nil {
		a.mcw.reportError("Failed to list tags", err)
		a.tags = make([]*app.CharacterTag, 0)
		return
	}
	a.tags = tags
	a.tagList.Refresh()
	if len(tags) > 0 {
		a.emptyTagsHint.Hide()
	} else {
		a.emptyTagsHint.Show()
	}
}
