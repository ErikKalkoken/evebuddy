package managecharacters

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxdialog "github.com/ErikKalkoken/fyne-kx/dialog"
	kmodal "github.com/ErikKalkoken/fyne-kx/modal"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type manageTags struct {
	widget.BaseWidget

	addCharactersButton *widget.Button
	characterList       *widget.List
	characters          []*app.EntityShort
	emptyCharactersHint fyne.CanvasObject
	emptyTagsHint       fyne.CanvasObject
	manageCharacters    *xwidget.AppBar
	cw                  *manageCharacters
	selectedTag         *app.CharacterTag
	tagList             *widget.List
	tags                []*app.CharacterTag
}

func newManageTags(cw *manageCharacters) *manageTags {
	a := &manageTags{
		cw: cw,
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
	})
	return a
}

func (a *manageTags) CreateRenderer() fyne.WidgetRenderer {
	// p := theme.Padding()
	addTag := widget.NewButtonWithIcon("Create tag", theme.ContentAddIcon(), func() {
		a.modifyTag("Create Character Tag", "Create", func(name string) error {
			_, err := a.cw.u.Character().CreateTag(context.Background(), name)
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
	actions := kxwidget.NewIconButtonWithMenu(theme.MoreHorizontalIcon(), fyne.NewMenu("",
		fyne.NewMenuItem("Save tags to file", a.exportTags),
		fyne.NewMenuItem("Replace tags from file", a.importTags),
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
	xdialog.ShowConfirm(
		"Delete All Tags",
		"Are you sure you want to delete all tags?",
		"Delete",
		func(confirmed bool) {
			if !confirmed {
				return
			}
			m := kmodal.NewProgressInfinite(
				"Deleting all tags",
				"Deleting...",
				func() error {
					ctx := context.Background()
					err := a.cw.u.Character().DeleteAllTags(ctx)
					if err != nil {
						return err
					}
					a.update(ctx)
					go a.cw.u.Signals().TagsChanged.Emit(ctx, struct{}{})
					return nil
				},
				a.cw.w,
			)
			m.OnError = func(err error) {
				fyne.Do(func() {
					a.cw.reportError("Failed to delete tags", err)
				})
			}
			m.Start()
		},
		a.cw.w,
	)
}

func (a *manageTags) exportTags() {
	d := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if writer == nil {
			return
		}
		m := kmodal.NewProgressInfinite(
			"Saving tags to file",
			"Saving...",
			func() error {
				defer writer.Close()
				if err != nil {
					return err
				}
				err := a.cw.u.Character().WriteTags(context.Background(), writer, fyne.CurrentApp().Metadata().Version)
				if err != nil {
					return err
				}
				slog.Info("Tags exported to file", "uri", writer.URI())
				a.cw.sb.Show("Tags exported")
				return nil
			},
			a.cw.w,
		)
		m.OnError = func(err error) {
			fyne.Do(func() {
				xdialog.ShowErrorAndLog("Failed to export tags", err, a.cw.u.IsDeveloperMode(), a.cw.w)
			})
		}
		m.Start()
	},
		a.cw.w,
	)
	kxdialog.AddDialogKeyHandler(d, a.cw.w)
	d.SetTitleText("Save tags to file")
	d.Show()
}

func (a *manageTags) importTags() {
	d := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		m := kmodal.NewProgressInfinite(
			"Replacing tags from file",
			"Replacing...",
			func() error {
				if reader == nil {
					return nil
				}
				defer reader.Close()
				if err != nil {
					return err
				}
				ctx := context.Background()
				err = a.cw.u.Character().ReadAndReplaceTags(ctx, reader, fyne.CurrentApp().Metadata().Version)
				if err != nil {
					return err
				}
				a.update(ctx)
				go a.cw.u.Signals().TagsChanged.Emit(ctx, struct{}{})
				slog.Info("Tags imported from file", "uri", reader.URI())
				return nil
			},
			a.cw.w,
		)
		m.OnError = func(err error) {
			fyne.Do(func() {
				xdialog.ShowErrorAndLog("Failed to import tags", err, a.cw.u.IsDeveloperMode(), a.cw.w)
			})
		}
		m.Start()
	},
		a.cw.w,
	)
	kxdialog.AddDialogKeyHandler(d, a.cw.w)
	d.SetTitleText("Replace tags from file")
	d.SetConfirmText("Replace")
	d.Show()
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
		_, others, err := a.cw.u.Character().ListCharactersForTag(context.Background(), a.selectedTag.ID)
		if err != nil {
			a.cw.reportError("Failed to list characters", err)
			return
		}
		if len(others) == 0 {
			return
		}
		selected := make(map[int64]bool)
		list := widget.NewList(
			func() int {
				return len(others)
			},
			func() fyne.CanvasObject {
				check := widget.NewIcon(theme.CheckButtonIcon())
				character := ui.NewEveEntityListItem(ui.LoadEveEntityIconFunc(a.cw.u.EVEImage()))
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
					err := a.cw.u.Character().AddTagToCharacter(
						context.Background(),
						characterID,
						a.selectedTag.ID,
					)
					if err != nil {
						a.cw.reportError("Failed to add tag to character", err)
						return
					}
				}
				a.setCharactersAsync(a.selectedTag)
				go a.cw.u.Signals().TagsChanged.Emit(context.Background(), struct{}{})
			},
			a.cw.w,
		)
		xdesktop.DisableShortcutsForDialog(d, a.cw.w)
		d.Show()
		_, s := a.cw.w.Canvas().InteractiveArea()
		d.Resize(fyne.NewSize(s.Width*0.8, s.Height*0.8))
	})
	w.Importance = widget.HighImportance
	w.Disable()
	return w
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
				a.modifyTag("Rename tag: "+tag.Name, "Rename", func(name string) error {
					return a.cw.u.Character().RenameTag(context.Background(), tag.ID, name)
				})
			}
			icons[1].(*ttwidget.Button).OnTapped = func() {
				s := "Are you sure you want to delete tag " + tag.Name + "?"
				xdialog.ShowConfirm(
					"Delete Tag", s, "Delete", func(confirmed bool) {
						if !confirmed {
							return
						}
						ctx := context.Background()
						err := a.cw.u.Character().DeleteTag(ctx, tag.ID)
						if err != nil {
							xdialog.ShowErrorAndLog("Failed to delete tag", err, a.cw.u.IsDeveloperMode(), a.cw.w)
							return
						}
						a.update(ctx)
						go a.cw.u.Signals().TagsChanged.Emit(ctx, struct{}{})
						if len(a.tags) > 0 {
							a.tagList.Select(0)
							return
						}
						a.tagList.UnselectAll()
						a.selectedTag = nil
						a.characters = xslices.Reset(a.characters)
						a.addCharactersButton.Disable()
						a.characterList.Refresh()
						a.addCharactersButton.Disable()
						a.manageCharacters.Hide()
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

func (a *manageTags) makeCharacterList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			remove := ttwidget.NewButtonWithIcon("", theme.CancelIcon(), nil)
			remove.SetToolTip("Remove character from tag")
			character := ui.NewEveEntityListItem(ui.LoadEveEntityIconFunc(a.cw.u.EVEImage()))
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
				err := a.cw.u.Character().RemoveTagFromCharacter(
					context.Background(),
					r.ID,
					a.selectedTag.ID,
				)
				if err != nil {
					a.cw.reportError("Failed to remove tag from character: "+a.selectedTag.Name, err)
					return
				}
				a.setCharactersAsync(a.selectedTag)
				go a.cw.u.Signals().TagsChanged.Emit(context.Background(), struct{}{})
			}
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(_ widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *manageTags) setCharactersAsync(tag *app.CharacterTag) {
	a.selectedTag = tag
	if tag == nil {
		a.characters = xslices.Reset(a.characters)
		a.manageCharacters.Hide()
		a.emptyCharactersHint.Show()
		return
	}
	a.manageCharacters.SetTitle("Tag: " + tag.Name)
	a.manageCharacters.Show()
	go func() {
		tagged, others, err := a.cw.u.Character().ListCharactersForTag(context.Background(), tag.ID)
		if err != nil {
			a.cw.reportError("Failed to list characters for "+tag.Name, err)
			return
		}
		fyne.Do(func() {
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
	}()
}

func (a *manageTags) modifyTag(title, confirm string, execute func(name string) error) {
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
				xdialog.ShowErrorAndLog("Failed to modify tag", err, a.cw.u.IsDeveloperMode(), a.cw.w)
				return
			}
			ctx := context.Background()
			a.update(ctx)
			go a.cw.u.Signals().TagsChanged.Emit(ctx, struct{}{})
		}, a.cw.w,
	)
	xdesktop.DisableShortcutsForDialog(d, a.cw.w)
	d.Show()
	d.Resize(fyne.NewSize(300, 200))
	a.cw.w.Canvas().Focus(name)
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
	tags, err := a.cw.u.Character().ListTagsByName(ctx)
	if err != nil {
		a.cw.reportError("Failed to list tags", err)
		a.tags = xslices.Reset(a.tags)
		return
	}
	fyne.Do(func() {
		a.tags = tags
		a.tagList.Refresh()
		a.tagList.UnselectAll()
		if len(tags) > 0 {
			a.emptyTagsHint.Hide()
			a.selectTagByName(tags[0].Name)
			a.setCharactersAsync(tags[0])
		} else {
			a.emptyTagsHint.Show()
			a.setCharactersAsync(nil)
		}
	})
}
