package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

const implantsUpdateTicker = 10 * time.Second

// implantsArea is the UI area that shows the skillqueue
type implantsArea struct {
	content   *fyne.Container
	implants  binding.UntypedList
	errorText binding.String
	top       *widget.Label
	ui        *ui
}

func (u *ui) NewImplantsArea() *implantsArea {
	a := implantsArea{
		implants:  binding.NewUntypedList(),
		errorText: binding.NewString(),
		top:       widget.NewLabel(""),
		ui:        u,
	}
	a.top.TextStyle.Bold = true
	list := widget.NewListWithData(
		a.implants,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.AccountIcon()),
				widget.NewLabel("template"),
			)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			// icon := row.Objects[0].(*widget.Label)
			name := row.Objects[1].(*widget.Label)

			q, err := convertDataItem[*model.CharacterImplant](di)
			if err != nil {
				slog.Error("failed to render row in implants table", "err", err)
				name.Text = "failed to render"
				name.Importance = widget.DangerImportance
				name.Refresh()
				return
			}
			name.SetText(q.EveType.Name)
		})

	list.OnSelected = func(id widget.ListItemID) {
		x, err := getFromBoundUntypedList[*model.CharacterImplant](a.implants, id)
		if err != nil {
			slog.Error("failed to access implant item in list", "err", err)
			return
		}
		var data = []struct {
			label string
			value string
			wrap  bool
		}{
			{"Name", x.EveType.Name, false},
			{"Description", x.EveType.Description, true},
		}
		form := widget.NewForm()
		for _, row := range data {
			c := widget.NewLabel(row.value)
			if row.wrap {
				c.Wrapping = fyne.TextWrapWord
			}
			form.Append(row.label, c)
		}
		s := container.NewScroll(form)
		dlg := dialog.NewCustom("Implant Detail", "OK", s, u.window)
		dlg.Show()
		dlg.Resize(fyne.Size{
			Width:  0.8 * a.ui.window.Canvas().Size().Width,
			Height: 0.8 * a.ui.window.Canvas().Size().Height,
		})
		list.UnselectAll()
	}

	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, list)
	return &a
}

func (a *implantsArea) Refresh() {
	err := a.updateData()
	if err != nil {
		slog.Error("failed to update skillqueue items for character", "characterID", a.ui.CurrentCharID(), "err", err)
		return
	}
	t, i := a.makeTopText()
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
}

func (a *implantsArea) updateData() error {
	if err := a.errorText.Set(""); err != nil {
		return err
	}
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		err := a.implants.Set(make([]any, 0))
		if err != nil {
			return err
		}
	}
	implants, err := a.ui.service.ListCharacterImplants(characterID)
	if err != nil {
		return err
	}
	items := make([]any, len(implants))
	for i, o := range implants {
		items[i] = o
	}
	if err := a.implants.Set(items); err != nil {
		panic(err)
	}
	return nil
}

func (a *implantsArea) makeTopText() (string, widget.Importance) {
	errorText, err := a.errorText.Get()
	if err != nil {
		panic(err)
	}
	if errorText != "" {
		return errorText, widget.DangerImportance
	}
	hasData, err := a.ui.service.CharacterSectionWasUpdated(a.ui.CurrentCharID(), model.CharacterSectionSkillqueue)
	if err != nil {
		return "ERROR", widget.DangerImportance
	}
	if !hasData {
		return "No data", widget.LowImportance
	}
	return fmt.Sprintf("%d implants", a.implants.Length()), widget.MediumImportance
}

func (a *implantsArea) StartUpdateTicker() {
	ticker := time.NewTicker(implantsUpdateTicker)
	go func() {
		for {
			func() {
				cc, err := a.ui.service.ListCharactersShort()
				if err != nil {
					slog.Error("Failed to fetch list of characters", "err", err)
					return
				}
				for _, c := range cc {
					a.MaybeUpdateAndRefresh(c.ID)
				}
			}()
			<-ticker.C
		}
	}()
}

func (a *implantsArea) MaybeUpdateAndRefresh(characterID int32) {
	changed, err := a.ui.service.UpdateCharacterSectionIfExpired(characterID, model.CharacterSectionImplants)
	if err != nil {
		slog.Error("Failed to update implants", "character", characterID, "err", err)
		return
	}
	if changed && characterID == a.ui.CurrentCharID() {
		a.Refresh()
	}
}