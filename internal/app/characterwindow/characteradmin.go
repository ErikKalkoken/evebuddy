package characterwindow

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/eveauth"
	kmodal "github.com/ErikKalkoken/fyne-kx/modal"
	"github.com/ErikKalkoken/go-set"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	awidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type characterAdminRow struct {
	characterID   int64
	corporationID int64
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
		mc: mc,
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
	a.ab.HideBackground = !a.mc.isMobile
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
			delete := ttwidget.NewButtonWithIcon("", theme.DeleteIcon(), func() {})
			delete.Importance = widget.DangerImportance
			delete.SetToolTip("Delete character")
			issueLabel := ttwidget.NewLabel("Missing scopes")
			issueLabel.Importance = widget.WarningImportance
			issueIcon := ttwidget.NewIcon(theme.NewWarningThemedResource(theme.WarningIcon()))
			issue := container.New(
				layout.NewCustomPaddedHBoxLayout(-p),
				issueIcon,
				issueLabel,
			)
			issue.Hide()
			row := container.NewBorder(
				nil,
				nil,
				nil,
				container.NewHBox(
					issue,
					layout.NewSpacer(),
					delete,
				),
				awidget.NewEntityListItem(true, a.mc.eis.CharacterPortraitAsync),
			)
			return row
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.characters) {
				return
			}
			r := a.characters[id]
			border := co.(*fyne.Container).Objects

			border[0].(*awidget.EntityListItem).Set(r.characterID, r.characterName)

			hbox := border[1].(*fyne.Container).Objects
			issueBox := hbox[0].(*fyne.Container)
			issueIcon := issueBox.Objects[0].(*ttwidget.Icon)
			issueLabel := issueBox.Objects[1].(*ttwidget.Label)
			if r.missingScopes.Size() != 0 {
				x := slices.Sorted(r.missingScopes.All())
				s := "Please re-add to approve missing scopes: " + strings.Join(x, ", ")
				issueIcon.SetToolTip(s)
				issueLabel.SetToolTip(s)
				issueBox.Show()
			} else {
				issueBox.Hide()
			}

			delete := hbox[2].(*ttwidget.Button)
			delete.OnTapped = func() {
				a.showDeleteDialog(r)
			}
		})

	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *characterAdmin) update(ctx context.Context) {
	characters, err := a.fetchRows(ctx)
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

func (a *characterAdmin) fetchRows(ctx context.Context) ([]characterAdminRow, error) {
	var rows []characterAdminRow
	cc, err := a.mc.cs.ListCharacters(ctx)
	if err != nil {
		return rows, err
	}
	for _, c := range cc {
		missing, err := a.mc.cs.MissingScopes(ctx, c.ID, app.Scopes())
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
			character, err := a.mc.cs.UpdateOrCreateCharacterFromSSO(cancelCTX, func(s string) {
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
			ctx := context.Background()
			a.update(ctx)
			if !a.mc.u.HasCharacter() {
				a.mc.u.LoadCharacter(character.ID)
			}
			if !a.mc.u.HasCorporation() {
				if c := character.EveCharacter.Corporation; !c.IsNPC().ValueOrZero() {
					a.mc.u.LoadCorporation(c.ID)
				}
			}
			go a.mc.signals.CharacterAdded.Emit(ctx, character)
			if !a.mc.isUpdateDisabled {
				go a.mc.u.UpdateCharacterAndRefreshIfNeeded(ctx, character.ID, true)
			}
			return nil
		}()
		if err != nil {
			fyne.Do(func() {
				d1.Hide()
				a.mc.u.ShowErrorDialog("Failed to add a new character", err, a.mc.w)
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
					wasCorpDeleted, err := a.mc.cs.DeleteCharacter(ctx, r.characterID)
					if err != nil {
						return err
					}
					a.update(ctx)
					if a.mc.u.CurrentCharacterID() == r.characterID {
						err := a.mc.u.SetAnyCharacter()
						if err != nil {
							slog.Error("delete character", "error", err)
							a.mc.sb.Show("Error: " + a.mc.u.HumanizeError(err))
						}
					}
					if wasCorpDeleted {
						err := a.mc.u.SetAnyCorporation()
						if err != nil {
							slog.Error("delete corporation", "error", err)
							a.mc.sb.Show("Error: " + a.mc.u.HumanizeError(err))
						}

					} else {
						ok, err := a.mc.rs.HasCorporation(ctx, r.corporationID)
						if err != nil {
							slog.Error("Failed to determine if corp exists", "err", err)
						}
						if ok {
							err := a.mc.rs.RemoveSectionDataWhenPermissionLost(ctx, r.corporationID)
							if err != nil {
								slog.Error(
									"Failed to remove corp data after character was deleted",
									slog.Int64("characterID", r.characterID),
									slog.Any("error", err))
							}
							go a.mc.u.UpdateCorporationAndRefreshIfNeeded(ctx, r.corporationID, true)
						}
					}
					go a.mc.signals.CharacterRemoved.Emit(ctx, &app.EntityShort{
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
