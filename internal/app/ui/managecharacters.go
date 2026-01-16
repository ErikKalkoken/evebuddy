package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/eveauth"
	kxdialog "github.com/ErikKalkoken/fyne-kx/dialog"
	kmodal "github.com/ErikKalkoken/fyne-kx/modal"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func showManageCharactersWindow(u *baseUI) {
	w, created, onClosed := u.getOrCreateWindowWithOnClosed("manage-characters", "Manage Characters")
	if !created {
		w.Show()
		return
	}
	mc := u.manageCharacters
	mc.setWindow(w)
	w.SetContent(fynetooltip.AddWindowToolTipLayer(mc, w.Canvas()))
	w.Resize(fyne.Size{Width: 700, Height: 500})
	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
		mc.unsetWindow()
	})
	w.SetCloseIntercept(func() {
		w.Close()
		fynetooltip.DestroyWindowToolTipLayer(w.Canvas())
	})
	w.Show()
}

type manageCharacters struct {
	widget.BaseWidget

	characterAdmin    *characterAdmin
	characterTags     *characterTags
	characterTraining *characterTraining
	sb                *iwidget.Snackbar
	u                 *baseUI
	w                 fyne.Window
}

func newManageCharacters(u *baseUI) *manageCharacters {
	a := &manageCharacters{
		sb: u.snackbar,
		u:  u,
		w:  u.MainWindow(),
	}
	a.ExtendBaseWidget(a)
	a.characterAdmin = newCharacterAdmin(a)
	a.characterTags = newCharacterTags(a)
	a.characterTraining = newCharacterTraining(a)
	return a
}

func (a *manageCharacters) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewAppTabs(
		container.NewTabItem("Characters", a.characterAdmin),
		container.NewTabItem("Tags", a.characterTags),
		container.NewTabItem("Training", a.characterTraining),
	)
	c.SetTabLocation(container.TabLocationLeading)
	return widget.NewSimpleRenderer(c)
}

func (a *manageCharacters) setWindow(w fyne.Window) {
	a.w = w
	a.sb = iwidget.NewSnackbar(w)
	a.sb.Start()
}

func (a *manageCharacters) unsetWindow() {
	a.sb.Stop()
}

func (a *manageCharacters) update() {
	a.characterAdmin.update()
	a.characterTags.update()
	a.characterTraining.update()
}

func (a *manageCharacters) reportError(text string, err error) {
	slog.Error(text, "error", err)
	a.sb.Show(fmt.Sprintf("ERROR: %s: %s", text, err))
}

type characterAdminRow struct {
	characterID   int32
	corporationID int32
	characterName string
	missingScopes set.Set[string]
}

// characterAdmin is a UI component for authorizing and removing EVE Online characters.
type characterAdmin struct {
	widget.BaseWidget

	ab         *iwidget.AppBar
	characters []characterAdminRow
	list       *widget.List
	mc         *manageCharacters
}

func newCharacterAdmin(mc *manageCharacters) *characterAdmin {
	a := &characterAdmin{
		characters: make([]characterAdminRow, 0),
		mc:         mc,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeCharacterList()
	add := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		a.showAddCharacterDialog()
	})
	add.Importance = widget.HighImportance
	if a.mc.u.IsOffline() {
		add.Disable()
	}
	a.ab = iwidget.NewAppBar("Characters", container.NewBorder(
		nil,
		container.NewVBox(add, newStandardSpacer()),
		nil,
		nil,
		a.list,
	))
	a.ab.HideBackground = !a.mc.u.isMobile
	return a
}

func (a *characterAdmin) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.ab)
}

func (a *characterAdmin) makeCharacterList() *widget.List {
	p := theme.Padding()
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
			issueLabel := ttwidget.NewLabel("Missing scopes")
			issueLabel.Importance = widget.WarningImportance
			issueIcon := ttwidget.NewIcon(theme.NewWarningThemedResource(theme.WarningIcon()))
			issueBox := container.New(
				layout.NewCustomPaddedHBoxLayout(-p),
				issueIcon,
				issueLabel,
			)
			issueBox.Hide()
			row := container.NewHBox(portrait, name, issueBox, layout.NewSpacer(), delete)
			return row
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.characters) {
				return
			}
			c := a.characters[id]
			row := co.(*fyne.Container).Objects

			portrait := row[0].(*canvas.Image)
			go a.mc.u.updateCharacterAvatar(c.characterID, func(r fyne.Resource) {
				fyne.Do(func() {
					portrait.Resource = r
					portrait.Refresh()
				})
			})

			name := row[1].(*widget.Label)
			name.SetText(c.characterName)

			issueBox := row[2].(*fyne.Container)
			issueIcon := issueBox.Objects[0].(*ttwidget.Icon)
			issueLabel := issueBox.Objects[1].(*ttwidget.Label)
			if c.missingScopes.Size() != 0 {
				x := slices.Sorted(c.missingScopes.All())
				s := "Please re-add to approve missing scopes: " + strings.Join(x, ", ")
				issueIcon.SetToolTip(s)
				issueLabel.SetToolTip(s)
				issueBox.Show()
			} else {
				issueBox.Hide()
			}

			delete := row[4].(*ttwidget.Button)
			delete.OnTapped = func() {
				a.showDeleteDialog(c)
			}
		})

	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *characterAdmin) update() {
	characters, err := a.fetchRows()
	if err != nil {
		a.mc.reportError("Failed to update characters", err)
		return
	}
	fyne.Do(func() {
		a.characters = characters
		a.list.Refresh()
		a.ab.SetTitle(fmt.Sprintf("Characters (%d)", len(characters)))
	})
}

func (a *characterAdmin) fetchRows() ([]characterAdminRow, error) {
	ctx := context.Background()
	rows := make([]characterAdminRow, 0)
	cc, err := a.mc.u.cs.ListCharacters(ctx)
	if err != nil {
		return rows, err
	}
	for _, c := range cc {
		missing, err := a.mc.u.cs.MissingScopes(ctx, c.ID, app.Scopes())
		if err != nil {
			return rows, err
		}
		r := characterAdminRow{
			characterID:   c.ID,
			corporationID: c.EveCharacter.Corporation.ID,
			characterName: c.EveCharacter.Name,
			missingScopes: missing,
		}
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *characterAdmin) showAddCharacterDialog() {
	cancelCTX, cancel := context.WithCancel(context.Background())
	infoText := widget.NewLabel(
		"Please follow instructions in your\nbrowser to add a new character.",
	)
	infoText.Alignment = fyne.TextAlignCenter
	var d1 dialog.Dialog
	closeButton := widget.NewButton("Cancel", func() {
		d1.Hide()
	})
	d1 = dialog.NewCustomWithoutButtons(
		"Add Character",
		container.NewBorder(
			nil,
			container.NewCenter(closeButton),
			nil,
			nil,
			infoText,
		),
		a.mc.w,
	)
	a.mc.u.ModifyShortcutsForDialog(d1, a.mc.w)
	done := make(chan struct{})
	d1.SetOnClosed(func() {
		cancel()
		<-done
	})
	d1.Show()
	go func() {
		err := func() error {
			character, err := a.mc.u.cs.UpdateOrCreateCharacterFromSSO(cancelCTX, func(s string) {
				fyne.Do(func() {
					infoText.SetText(s)
					closeButton.Hide()
				})
			})
			if errors.Is(err, eveauth.ErrAborted) {
				return nil
			}
			if err != nil {
				return err
			}
			fyne.Do(func() {
				infoText.SetText("Adding new character...")
			})
			a.update()
			if !a.mc.u.hasCharacter() {
				a.mc.u.loadCharacter(character.ID)
			}
			if !a.mc.u.hasCorporation() {
				if c := character.EveCharacter.Corporation; !c.IsNPC().ValueOrZero() {
					a.mc.u.loadCorporation(c.ID)
				}
			}
			go a.mc.u.characterAdded.Emit(context.Background(), character)
			if !a.mc.u.isUpdateDisabled {
				go a.mc.u.updateCharacterAndRefreshIfNeeded(context.Background(), character.ID, true)
			}
			return nil
		}()
		if err != nil {
			fyne.Do(func() {
				d1.Hide()
				a.mc.u.showErrorDialog("Failed to add a new character", err, a.mc.w)
			})
		} else {
			fyne.Do(func() {
				d1.Hide()
			})

		}
		close(done)
	}()
}

func (a *characterAdmin) showDeleteDialog(r characterAdminRow) {
	a.mc.u.ShowConfirmDialog(
		"Delete Character",
		fmt.Sprintf("Are you sure you want to delete %s with all it's locally stored data?", r.characterName),
		"Delete",
		func(confirmed bool) {
			if !confirmed {
				return
			}
			m := kmodal.NewProgressInfinite(
				"Deleting character",
				fmt.Sprintf("Deleting %s...", r.characterName),
				func() error {
					ctx := context.Background()
					corpDeleted, err := a.mc.u.cs.DeleteCharacter(ctx, r.characterID)
					if err != nil {
						return err
					}
					a.update()
					if a.mc.u.currentCharacterID() == r.characterID {
						a.mc.u.setAnyCharacter()
					}
					if corpDeleted {
						a.mc.u.setAnyCorporation()
					} else {
						ok, err := a.mc.u.rs.HasCorporation(ctx, r.corporationID)
						if err != nil {
							slog.Error("Failed to determine if corp exists", "err", err)
						}
						if ok {
							if err := a.mc.u.rs.RemoveSectionDataWhenPermissionLost(ctx, r.corporationID); err != nil {
								slog.Error("Failed to remove corp data after character was deleted", "characterID", r.characterID, "error", err)
							}
							go a.mc.u.updateCorporationAndRefreshIfNeeded(ctx, r.corporationID, true)
						}
					}
					go a.mc.u.characterRemoved.Emit(context.Background(), &app.EntityShort[int32]{
						ID:   r.characterID,
						Name: r.characterName,
					})
					return nil
				},
				a.mc.w,
			)
			m.OnSuccess = func() {
				a.mc.sb.Show(fmt.Sprintf("Character %s deleted", r.characterName))
			}
			m.OnError = func(err error) {
				a.mc.reportError(fmt.Sprintf("ERROR: Failed to delete character %s", r.characterName), err)
			}
			m.Start()
		},
		a.mc.w,
	)
}

type characterTags struct {
	widget.BaseWidget

	addCharactersButton *widget.Button
	characterList       *widget.List
	characters          []*app.EntityShort[int32]
	emptyCharactersHint fyne.CanvasObject
	emptyTagsHint       fyne.CanvasObject
	manageCharacters    *iwidget.AppBar
	mc                  *manageCharacters
	selectedTag         *app.CharacterTag
	tagList             *widget.List
	tags                []*app.CharacterTag
}

func newCharacterTags(mc *manageCharacters) *characterTags {
	a := &characterTags{
		characters: make([]*app.EntityShort[int32], 0),
		mc:         mc,
		tags:       make([]*app.CharacterTag, 0),
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
	a.mc.u.characterRemoved.AddListener(func(ctx context.Context, c *app.EntityShort[int32]) {
		a.update()
		a.mc.u.tagsChanged.Emit(ctx, struct{}{})
	})
	return a
}

func (a *characterTags) CreateRenderer() fyne.WidgetRenderer {
	// p := theme.Padding()
	addTag := widget.NewButtonWithIcon("Create tag", theme.ContentAddIcon(), func() {
		a.modifyTag("Create Character Tag", "Create", func(name string) error {
			_, err := a.mc.u.cs.CreateTag(context.Background(), name)
			return err
		})
	})
	addTag.Importance = widget.HighImportance
	main := container.NewBorder(
		nil,
		container.NewVBox(addTag, newStandardSpacer()),
		nil,
		nil,
		a.tagList,
	)
	actions := kxwidget.NewIconButtonWithMenu(theme.MoreHorizontalIcon(), fyne.NewMenu("",
		fyne.NewMenuItem("Save tags to file", a.exportTags),
		fyne.NewMenuItem("Replace tags from file", a.importTags),
		fyne.NewMenuItem("Delete all tags", a.deleteTags),
	))
	ab := iwidget.NewAppBar("Tags", main, actions)
	ab.HideBackground = !a.mc.u.isMobile
	c := container.NewVSplit(
		container.NewStack(ab, a.emptyTagsHint),
		container.NewStack(a.manageCharacters, a.emptyCharactersHint),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *characterTags) deleteTags() {
	a.mc.u.ShowConfirmDialog(
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
					err := a.mc.u.cs.DeleteAllTags(ctx)
					if err != nil {
						return err
					}
					a.update()
					go a.mc.u.tagsChanged.Emit(ctx, struct{}{})
					return nil
				},
				a.mc.w,
			)
			m.OnError = func(err error) {
				fyne.Do(func() {
					a.mc.reportError("Failed to delete tags", err)
				})
			}
			m.Start()
		},
		a.mc.w,
	)
}

func (a *characterTags) exportTags() {
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
				err := a.mc.u.cs.WriteTags(context.Background(), writer, a.mc.u.app.Metadata().Version)
				if err != nil {
					return err
				}
				slog.Info("Tags exported to file", "uri", writer.URI())
				a.mc.sb.Show("Tags exported")
				return nil
			},
			a.mc.w,
		)
		m.OnError = func(err error) {
			fyne.Do(func() {
				a.mc.u.showErrorDialog("Failed to export tags", err, a.mc.w)
			})
		}
		m.Start()
	},
		a.mc.w,
	)
	kxdialog.AddDialogKeyHandler(d, a.mc.w)
	d.SetTitleText("Save tags to file")
	d.Show()
}

func (a *characterTags) importTags() {
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
				err = a.mc.u.cs.ReadAndReplaceTags(ctx, reader, a.mc.u.app.Metadata().Version)
				if err != nil {
					return err
				}
				a.update()
				go a.mc.u.tagsChanged.Emit(ctx, struct{}{})
				slog.Info("Tags imported from file", "uri", reader.URI())
				return nil
			},
			a.mc.w,
		)
		m.OnError = func(err error) {
			fyne.Do(func() {
				a.mc.u.showErrorDialog("Failed to import tags", err, a.mc.w)
			})
		}
		m.Start()
	},
		a.mc.w,
	)
	kxdialog.AddDialogKeyHandler(d, a.mc.w)
	d.SetTitleText("Replace tags from file")
	d.SetConfirmText("Replace")
	d.Show()
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
	ab.HideBackground = !a.mc.u.isMobile
	ab.Hide()
	return ab
}

func (a *characterTags) makeAddCharacterButton() *widget.Button {
	w := widget.NewButtonWithIcon("Add characters to tag", theme.ContentAddIcon(), func() {
		if a.selectedTag == nil {
			return
		}
		_, others, err := a.mc.u.cs.ListCharactersForTag(context.Background(), a.selectedTag.ID)
		if err != nil {
			a.mc.reportError("Failed to list characters", err)
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
				go a.mc.u.updateCharacterAvatar(character.ID, func(r fyne.Resource) {
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
					err := a.mc.u.cs.AddTagToCharacter(
						context.Background(),
						characterID,
						a.selectedTag.ID,
					)
					if err != nil {
						a.mc.reportError("Failed to add tag to character", err)
						return
					}
				}
				a.setCharacters(a.selectedTag)
				go a.mc.u.tagsChanged.Emit(context.Background(), struct{}{})
			},
			a.mc.w,
		)
		a.mc.u.ModifyShortcutsForDialog(d, a.mc.w)
		d.Show()
		_, s := a.mc.w.Canvas().InteractiveArea()
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
					return a.mc.u.cs.RenameTag(context.Background(), tag.ID, name)
				})
			}
			icons[1].(*ttwidget.Button).OnTapped = func() {
				s := "Are you sure you want to delete tag " + tag.Name + "?"
				a.mc.u.ShowConfirmDialog(
					"Delete Tag", s, "Delete", func(confirmed bool) {
						if !confirmed {
							return
						}
						err := a.mc.u.cs.DeleteTag(context.Background(), tag.ID)
						if err != nil {
							a.mc.u.showErrorDialog("Failed to delete tag", err, a.mc.w)
							return
						}
						a.update()
						go a.mc.u.tagsChanged.Emit(context.Background(), struct{}{})
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
					}, a.mc.w,
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
		a.setCharacters(tag)
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
			go a.mc.u.updateCharacterAvatar(character.ID, func(r fyne.Resource) {
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
				err := a.mc.u.cs.RemoveTagFromCharacter(
					context.Background(),
					character.ID,
					a.selectedTag.ID,
				)
				if err != nil {
					a.mc.reportError("Failed to remove tag from character: "+a.selectedTag.Name, err)
					return
				}
				a.setCharacters(a.selectedTag)
				go a.mc.u.tagsChanged.Emit(context.Background(), struct{}{})
			}
		},
	)
	l.HideSeparators = true
	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *characterTags) setCharacters(tag *app.CharacterTag) {
	a.selectedTag = tag
	if tag == nil {
		a.characters = make([]*app.EntityShort[int32], 0)
		a.manageCharacters.Hide()
		a.emptyCharactersHint.Show()
		return
	}
	a.manageCharacters.SetTitle("Tag: " + tag.Name)
	a.manageCharacters.Show()
	tagged, others, err := a.mc.u.cs.ListCharactersForTag(context.Background(), tag.ID)
	if err != nil {
		a.mc.reportError("Failed to list characters for "+tag.Name, err)
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
				a.mc.u.showErrorDialog("Failed to modify tag", err, a.mc.w)
				return
			}
			a.update()
			go a.mc.u.tagsChanged.Emit(context.Background(), struct{}{})
		}, a.mc.w,
	)
	a.mc.u.ModifyShortcutsForDialog(d, a.mc.w)
	d.Show()
	d.Resize(fyne.NewSize(300, 200))
	a.mc.w.Canvas().Focus(name)
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

func (a *characterTags) update() {
	tags, err := a.mc.u.cs.ListTagsByName(context.Background())
	if err != nil {
		a.mc.reportError("Failed to list tags", err)
		a.tags = make([]*app.CharacterTag, 0)
		return
	}
	fyne.Do(func() {
		a.tags = tags
		a.tagList.Refresh()
		a.tagList.UnselectAll()
		if len(tags) > 0 {
			a.emptyTagsHint.Hide()
			a.selectTagByName(tags[0].Name)
			a.setCharacters(tags[0])
		} else {
			a.emptyTagsHint.Show()
			a.setCharacters(nil)
		}
	})
}

// characterTraining is a UI component that allows to configure training watchers for characters.
type characterTraining struct {
	widget.BaseWidget

	characters []*app.Character
	list       *widget.List
	mc         *manageCharacters
}

func newCharacterTraining(mc *manageCharacters) *characterTraining {
	a := &characterTraining{
		mc: mc,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeList()

	// Signals
	a.mc.u.characterAdded.AddListener(func(_ context.Context, _ *app.Character) {
		a.update()
	})
	a.mc.u.characterRemoved.AddListener(func(_ context.Context, _ *app.EntityShort[int32]) {
		a.update()
	})
	return a
}

func (a *characterTraining) CreateRenderer() fyne.WidgetRenderer {
	actions := kxwidget.NewIconButtonWithMenu(theme.MoreHorizontalIcon(), fyne.NewMenu("",
		fyne.NewMenuItem("Set to currently trained", func() {
			go func() {
				ctx := context.Background()
				for id, c := range a.characters {
					d, err := a.mc.u.cs.TotalTrainingTime(ctx, c.ID)
					if err != nil {
						slog.Error("Failed to set watcher for trained characters", "error", err)
						continue
					}
					fyne.Do(func() {
						a.updateCharacterWatched(ctx, id, d.ValueOrZero() > 0)
					})
				}
			}()
		}),
		fyne.NewMenuItem("Enable all", func() {
			ctx := context.Background()
			for id := range a.characters {
				a.updateCharacterWatched(ctx, id, true)
			}
		}),
		fyne.NewMenuItem("Disable all", func() {
			ctx := context.Background()
			for id := range a.characters {
				a.updateCharacterWatched(ctx, id, false)
			}
		}),
	))
	ab := iwidget.NewAppBar("Watched Training", a.list, actions)
	ab.HideBackground = !a.mc.u.isMobile
	return widget.NewSimpleRenderer(ab)
}

func (a *characterTraining) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			portrait := iwidget.NewImageFromResource(
				icons.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			character := widget.NewLabel("Character")
			character.Truncation = fyne.TextTruncateEllipsis
			return container.NewBorder(
				nil,
				nil,
				portrait,
				kxwidget.NewSwitch(nil),
				character,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.characters) {
				return
			}
			c := a.characters[id]
			row := co.(*fyne.Container).Objects

			character := row[0].(*widget.Label)
			character.SetText(c.EveCharacter.Name)

			portrait := row[1].(*canvas.Image)
			go a.mc.u.updateCharacterAvatar(c.ID, func(r fyne.Resource) {
				fyne.Do(func() {
					portrait.Resource = r
					portrait.Refresh()
				})
			})

			sw := row[2].(*kxwidget.Switch)
			sw.On = c.IsTrainingWatched
			sw.Refresh()
			sw.OnChanged = func(on bool) {
				a.updateCharacterWatched(context.Background(), id, on)
			}
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.characters) {
			return
		}
		c := a.characters[id]
		v := !c.IsTrainingWatched
		a.updateCharacterWatched(context.Background(), id, v)
	}
	return l
}

func (a *characterTraining) updateCharacterWatched(ctx context.Context, id int, on bool) {
	if id >= len(a.characters) {
		return
	}
	c := a.characters[id]
	go func() {
		err := a.mc.u.cs.UpdateIsTrainingWatched(ctx, c.ID, on)
		if err != nil {
			slog.Error("Failed to update training watcher", "characterID", c.ID, "error", err)
			a.mc.u.ShowSnackbar("Failed to update training watcher: " + a.mc.u.humanizeError(err))
		}
		fyne.Do(func() {
			a.characters[id].IsTrainingWatched = on
			a.list.RefreshItem(id)
		})
		a.mc.u.characterChanged.Emit(ctx, c.ID)
	}()
}

func (a *characterTraining) update() {
	characters, err := a.mc.u.cs.ListCharacters(context.Background())
	if err != nil {
		panic(err)
	}
	slices.SortFunc(characters, func(a, b *app.Character) int {
		return strings.Compare(a.EveCharacter.Name, b.EveCharacter.Name)
	})
	fyne.Do(func() {
		a.characters = characters
		a.list.Refresh()
	})
}
