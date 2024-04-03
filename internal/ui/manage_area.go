package ui

import (
	"context"
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/api/sso"
	"example/esiapp/internal/model"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var esiScopes = []string{
	"esi-characters.read_contacts.v1",
	"esi-mail.read_mail.v1",
	"esi-mail.organize_mail.v1",
	"esi-mail.send_mail.v1",
	"esi-search.search_structures.v1",
}

// manageArea is the UI area for managing of characters.
type manageArea struct {
	content *fyne.Container
	dialog  *dialog.CustomDialog
	ui      *ui
}

func (u *ui) ShowManageDialog() {
	m := u.NewManageArea()
	m.Redraw()
	button := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		m.showAddCharacterDialog()
	})
	button.Importance = widget.HighImportance
	c := container.NewScroll(m.content)
	c.SetMinSize(fyne.NewSize(400, 400))
	content := container.NewBorder(button, nil, nil, nil, c)
	dialog := dialog.NewCustom("Manage Characters", "Close", content, u.window)
	m.dialog = dialog
	dialog.SetOnClosed(func() {
		u.characterArea.Redraw()
	})
	dialog.Show()
}

func (u *ui) NewManageArea() *manageArea {
	content := container.NewVBox()
	m := &manageArea{
		ui:      u,
		content: content,
	}
	return m
}

func (m *manageArea) Redraw() {
	chars, err := model.FetchAllCharacters()
	if err != nil {
		panic(err)
	}
	m.content.RemoveAll()
	for _, char := range chars {
		uri := char.PortraitURL(defaultIconSize)
		image := canvas.NewImageFromURI(uri)
		image.FillMode = canvas.ImageFillOriginal
		name := widget.NewLabel(char.Name)
		selectButton := widget.NewButtonWithIcon("Select", theme.ConfirmIcon(), func() {
			m.ui.SetCurrentCharacter(&char)
			m.dialog.Hide()
		})
		isCurrentChar := char.ID == m.ui.CurrentCharID()
		if isCurrentChar {
			selectButton.Disable()
		}
		deleteButton := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
			dialog := dialog.NewConfirm(
				"Delete Character",
				fmt.Sprintf("Are you sure you want to delete %s?", char.Name),
				func(confirmed bool) {
					if confirmed {
						err := char.Delete()
						if err != nil {
							d := dialog.NewError(err, m.ui.window)
							d.Show()
						}
						m.Redraw()
						if isCurrentChar {
							m.ui.ResetCurrentCharacter()
							m.ui.characterArea.Redraw()
						}
					}
				},
				m.ui.window,
			)
			dialog.Show()
		})
		deleteButton.Importance = widget.DangerImportance
		item := container.NewHBox(image, name, layout.NewSpacer(), selectButton, deleteButton)
		m.content.Add(item)
		m.content.Add(widget.NewSeparator())
	}
	m.content.Refresh()
}

func (m *manageArea) showAddCharacterDialog() {
	ctx, cancel := context.WithCancel(context.Background())
	dialog := dialog.NewCustom(
		"Add Character",
		"Cancel",
		widget.NewLabel("Please follow instructions in your browser to add a new character."),
		m.ui.window,
	)
	dialog.SetOnClosed(cancel)
	go func() {
		defer cancel()
		defer dialog.Hide()
		_, err := addCharacter(ctx)
		if err != nil {
			slog.Error("Failed to add a new character", "error", err)
		} else {
			m.Redraw()
		}
	}()
	dialog.Show()
}

// addCharacter adds a new character via SSO authentication and returns the new token.
func addCharacter(ctx context.Context) (*model.Token, error) {
	ssoToken, err := sso.Authenticate(ctx, httpClient, esiScopes)
	if err != nil {
		return nil, err
	}
	charID := ssoToken.CharacterID
	charEsi, err := esi.FetchCharacter(httpClient, charID)
	if err != nil {
		return nil, err
	}
	ids := []int32{charID, charEsi.CorporationID}
	if charEsi.AllianceID != 0 {
		ids = append(ids, charEsi.AllianceID)
	}
	if charEsi.FactionID != 0 {
		ids = append(ids, charEsi.FactionID)
	}
	_, err = AddMissingEveEntities(ids)
	if err != nil {
		return nil, err
	}
	character := model.Character{
		ID:            charID,
		Name:          charEsi.Name,
		CorporationID: charEsi.CorporationID,
	}
	if err = character.Save(); err != nil {
		return nil, err
	}
	token := model.Token{
		AccessToken:  ssoToken.AccessToken,
		Character:    character,
		ExpiresAt:    ssoToken.ExpiresAt,
		RefreshToken: ssoToken.RefreshToken,
		TokenType:    ssoToken.TokenType,
	}
	if err = token.Save(); err != nil {
		return nil, err
	}
	return &token, nil
}
