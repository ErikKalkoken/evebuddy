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
	"github.com/ErikKalkoken/evebuddy/internal/app/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/app/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type adminRow struct {
	characterID   int64
	corporationID int64
	characterName string
	missingScopes set.Set[string]
}

// admin is a UI component for authorizing and removing EVE Online characters.
type admin struct {
	widget.BaseWidget

	ab         *xwidget.AppBar
	cw         *characterWindow
	characters *widget.List
	rows       []adminRow
}

func newAdmin(cw *characterWindow) *admin {
	a := &admin{
		cw: cw,
	}
	a.ExtendBaseWidget(a)
	a.characters = a.makeCharacterList()
	add := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		a.showAddCharacterDialog()
	})
	add.Importance = widget.HighImportance
	if a.cw.u.IsOffline() {
		add.Disable()
	}
	a.ab = xwidget.NewAppBar("Characters", container.NewBorder(
		nil,
		container.NewVBox(add, xwidget.NewStandardSpacer()),
		nil,
		nil,
		a.characters,
	))
	a.ab.HideBackground = !a.cw.u.IsMobile()
	return a
}

func (a *admin) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.ab)
}

func (a *admin) makeCharacterList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rows)
		},
		func() fyne.CanvasObject {
			return newAdminListItem(a.showDeleteDialog, awidget.LoadEveEntityIconFunc(a.cw.u.EVEImage()))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rows) {
				return
			}
			co.(*adminListItem).set(a.rows[id])
		})

	l.OnSelected = func(_ widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *admin) update(ctx context.Context) {
	characters, err := a.fetchRows(ctx)
	if err != nil {
		a.cw.reportError("Failed to update characters", err)
		return
	}
	fyne.Do(func() {
		a.rows = characters
		a.characters.Refresh()
		a.ab.SetTitle(fmt.Sprintf("Characters (%d)", len(characters)))
	})
}

func (a *admin) fetchRows(ctx context.Context) ([]adminRow, error) {
	var rows []adminRow
	cc, err := a.cw.u.Character().ListCharacters(ctx)
	if err != nil {
		return rows, err
	}
	for _, c := range cc {
		missing, err := a.cw.u.Character().MissingScopes(ctx, c.ID, app.Scopes())
		if err != nil {
			return rows, err
		}
		r := adminRow{
			characterID:   c.ID,
			corporationID: c.EveCharacter.Corporation.ID,
			characterName: c.EveCharacter.Name,
			missingScopes: missing,
		}
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *admin) showAddCharacterDialog() {
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
		a.cw.w,
	)
	xdesktop.DisableShortcutsForDialog(d1, a.cw.w)
	done := make(chan struct{})
	d1.SetOnClosed(func() {
		cancel()
		<-done
	})
	d1.Show()
	go func() {
		err := func() error {
			character, err := a.cw.u.Character().UpdateOrCreateCharacterFromSSO(cancelCTX, func(s string) {
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
			if a.cw.u.CurrentCharacter() == nil {
				err := a.cw.u.LoadCharacter(ctx, character.ID)
				if err != nil {
					slog.Error(
						"Setting newly added character",
						slog.Int64("characterID", character.ID),
						slog.Any("error", err),
					)
					a.cw.sb.Show("Failed to load character")
				}
			}
			if a.cw.u.CurrentCorporation() == nil {
				if c := character.EveCharacter.Corporation; !c.IsNPC().ValueOrZero() {
					err := a.cw.u.LoadCorporation(ctx, c.ID)
					if err != nil {
						slog.Error(
							"Setting newly added corporation",
							slog.Int64("corporationID", c.ID),
							slog.Any("error", err),
						)
						a.cw.sb.Show("Failed to load corporation")
					}
				}
			}
			go a.cw.u.Signals().CharacterAdded.Emit(ctx, character)
			if !a.cw.u.IsUpdateDisabled() {
				go a.cw.u.Character().UpdateCharacterAndRefreshIfNeeded(ctx, character.ID, true)
			}
			return nil
		}()
		if err != nil {
			fyne.Do(func() {
				d1.Hide()
				xdialog.ShowErrorAndLog("Failed to add a new character", err, a.cw.u.IsDeveloperMode(), a.cw.w)
			})
		} else {
			fyne.Do(func() {
				d1.Hide()
			})

		}
		close(done)
	}()
}

func (a *admin) showDeleteDialog(r adminRow) {
	xdialog.ShowConfirm(
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
					wasCorpDeleted, err := a.cw.u.Character().DeleteCharacter(ctx, r.characterID)
					if err != nil {
						return err
					}
					a.update(ctx)
					if a.cw.u.CurrentCharacter().IDOrZero() == r.characterID {
						err := a.cw.u.SetAnyCharacter(ctx)
						if err != nil {
							slog.Error("delete character", "error", err)
							a.cw.sb.Show("Error: " + a.cw.u.ErrorDisplay(err))
						}
					}
					if wasCorpDeleted {
						err := a.cw.u.SetAnyCorporation(ctx)
						if err != nil {
							slog.Error("delete corporation", "error", err)
							a.cw.sb.Show("Error: " + a.cw.u.ErrorDisplay(err))
						}

					} else {
						ok, err := a.cw.u.Corporation().HasCorporation(ctx, r.corporationID)
						if err != nil {
							slog.Error("Failed to determine if corp exists", "err", err)
						}
						if ok {
							err := a.cw.u.Corporation().RemoveSectionDataWhenPermissionLost(ctx, r.corporationID)
							if err != nil {
								slog.Error(
									"Failed to remove corp data after character was deleted",
									slog.Int64("characterID", r.characterID),
									slog.Any("error", err))
							}
							go a.cw.u.Corporation().UpdateCorporationAndRefreshIfNeeded(ctx, r.corporationID, true)
						}
					}
					go a.cw.u.Signals().CharacterRemoved.Emit(ctx, &app.EntityShort{
						ID:   r.characterID,
						Name: r.characterName,
					})
					return nil
				},
				a.cw.w,
			)
			m.OnSuccess = func() {
				a.cw.sb.Show(fmt.Sprintf("Character %s deleted", r.characterName))
			}
			m.OnError = func(err error) {
				a.cw.reportError(fmt.Sprintf("ERROR: Failed to delete character %s", r.characterName), err)
			}
			m.Start()
		},
		a.cw.w,
	)
}

type adminListItem struct {
	widget.BaseWidget

	delete           *ttwidget.Button
	entityItem       *awidget.EveEntityListItem
	issue            *fyne.Container
	issueIcon        *ttwidget.Icon
	issueLabel       *ttwidget.Label
	showDeleteDialog func(adminRow)
}

func newAdminListItem(showDeleteDialog func(adminRow), loadIcon awidget.EveEntityIconLoader) *adminListItem {
	p := theme.Padding()
	del := ttwidget.NewButtonWithIcon("", theme.DeleteIcon(), func() {})
	del.Importance = widget.DangerImportance
	del.SetToolTip("Delete character")
	issueLabel := ttwidget.NewLabel("Missing scopes")
	issueLabel.Importance = widget.WarningImportance
	issueIcon := ttwidget.NewIcon(theme.NewWarningThemedResource(theme.WarningIcon()))
	issue := container.New(
		layout.NewCustomPaddedHBoxLayout(-p),
		issueIcon,
		issueLabel,
	)
	issue.Hide()
	character := awidget.NewEveEntityListItem(loadIcon)
	character.IsAvatar = true
	w := &adminListItem{
		delete:           del,
		entityItem:       character,
		issue:            issue,
		issueIcon:        issueIcon,
		issueLabel:       issueLabel,
		showDeleteDialog: showDeleteDialog,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *adminListItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(
			w.issue,
			layout.NewSpacer(),
			w.delete,
		),
		w.entityItem,
	)
	return widget.NewSimpleRenderer(c)
}

func (w *adminListItem) set(r adminRow) {
	w.entityItem.Set2(r.characterID, r.characterName, app.EveEntityCharacter)
	if r.missingScopes.Size() != 0 {
		x := slices.Sorted(r.missingScopes.All())
		s := "Please re-add to approve missing scopes: " + strings.Join(x, ", ")
		w.issueIcon.SetToolTip(s)
		w.issueLabel.SetToolTip(s)
		w.issue.Show()
	} else {
		w.issue.Hide()
	}

	w.delete.OnTapped = func() {
		w.showDeleteDialog(r)
	}
}
