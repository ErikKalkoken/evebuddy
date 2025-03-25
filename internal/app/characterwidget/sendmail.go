package characterwidget

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const labelWith = 45

type SendMail struct {
	widget.BaseWidget

	body      *widget.Entry
	character *app.Character
	from      *EveEntityEntry
	subject   *widget.Entry
	to        *EveEntityEntry
	u         app.UI
	w         fyne.Window
}

func NewSendMail(u app.UI, c *app.Character, mode app.SendMailMode, m *app.CharacterMail) *SendMail {
	a := &SendMail{
		character: c,
		u:         u,
		w:         u.MainWindow(),
	}
	a.ExtendBaseWidget(a)

	a.from = NewEveEntityEntry(widget.NewLabel("From"), labelWith, u.EveImageService())
	a.from.ShowInfoWindow = u.ShowEveEntityInfoWindow
	a.from.Set([]*app.EveEntity{{ID: c.ID, Name: c.EveCharacter.Name, Category: app.EveEntityCharacter}})
	a.from.Disable()

	toButton := widget.NewButton("To", func() {
		showAddDialog(u, c.ID, func(ee *app.EveEntity) {
			a.to.Add(ee)
		}, a.w)
	})
	a.to = NewEveEntityEntry(toButton, labelWith, u.EveImageService())
	a.to.ShowInfoWindow = u.ShowEveEntityInfoWindow
	a.to.Placeholder = "Tap To-Button to add recipients..."

	a.subject = widget.NewEntry()
	a.subject.PlaceHolder = "Subject"

	a.body = widget.NewEntry()
	a.body.MultiLine = true
	a.body.SetMinRowsVisible(14)
	a.body.PlaceHolder = "Compose message"

	if m != nil {
		const sep = "\n\n--------------------------------\n"
		switch mode {
		case app.SendMailReply:
			a.to.Set([]*app.EveEntity{m.From})
			a.subject.SetText(fmt.Sprintf("Re: %s", m.Subject))
			a.body.SetText(sep + m.String())
		case app.SendMailReplyAll:
			oo := slices.Concat([]*app.EveEntity{m.From}, m.Recipients)
			oo = slices.DeleteFunc(oo, func(o *app.EveEntity) bool {
				return o.ID == c.EveCharacter.ID
			})
			a.to.Set(oo)
			a.subject.SetText(fmt.Sprintf("Re: %s", m.Subject))
			a.body.SetText(sep + m.String())
		case app.SendMailForward:
			a.subject.SetText(fmt.Sprintf("Fw: %s", m.Subject))
			a.body.SetText(sep + m.String())
		default:
			panic(fmt.Errorf("undefined mode for create message: %v", mode))
		}
	}
	return a
}

func (a *SendMail) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(a.from, a.to, a.subject),
		nil,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *SendMail) SetWindow(w fyne.Window) {
	a.w = w
}

// sendAction tries to send the current mail and reports whether it was successful
func (a *SendMail) SendAction() bool {
	showErrorDialog := func(message string) {
		a.u.ShowInformationDialog("Failed to send mail", message, a.u.MainWindow())
	}
	if a.to.IsEmpty() {
		showErrorDialog("A mail needs to have at least one recipient.")
		return false
	}
	if a.subject.Text == "" {
		showErrorDialog("The subject can not be empty.")
		return false
	}
	if a.body.Text == "" {
		showErrorDialog("The message can not be empty.")
		return false
	}
	ctx := context.Background()
	_, err := a.u.CharacterService().SendCharacterMail(
		ctx,
		a.character.ID,
		a.subject.Text,
		a.to.Items(),
		a.body.Text,
	)
	if err != nil {
		showErrorDialog(err.Error())
		return false
	}
	a.u.ShowSnackbar(fmt.Sprintf("Your mail to %s has been sent.", a.to))
	return true
}

// func MakeSendMailPage(
// 	u app.UI,
// 	character *app.Character,
// 	mode app.SendMailMode,
// 	mail *app.CharacterMail,
// 	w fyne.Window,
// ) (fyne.CanvasObject, fyne.Resource, func() bool) {

// }

func showAddDialog(u app.UI, characterID int32, onSelected func(ee *app.EveEntity), w fyne.Window) {
	var modal *widget.PopUp
	results := make([]*app.EveEntity, 0)
	fallbackIcon := icons.Questionmark32Png
	list := widget.NewList(
		func() int {
			return len(results)
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("Template")
			name.Truncation = fyne.TextTruncateClip
			category := iwidget.NewLabelWithSize("Template", theme.SizeNameCaptionText)
			icon := iwidget.NewImageFromResource(icons.Questionmark32Png, fyne.NewSquareSize(app.IconUnitSize))
			return container.NewBorder(
				nil,
				nil,
				icon,
				category,
				name,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(results) {
				return
			}
			ee := results[id]
			row := co.(*fyne.Container).Objects
			row[0].(*widget.Label).SetText(ee.Name)
			image := row[1].(*canvas.Image)
			appwidget.RefreshImageResourceAsync(image, func() (fyne.Resource, error) {
				res, err := FetchEveEntityAvatar(u.EveImageService(), ee, fallbackIcon)
				if err != nil {
					return fallbackIcon, err
				}
				return res, nil
			})
			row[2].(*iwidget.Label).SetText(ee.CategoryDisplay())
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(results) {
			list.UnselectAll()
			return
		}
		onSelected(results[id])
		modal.Hide()
	}
	showErrorDialog := func(search string, err error) {
		slog.Error("Failed to resolve names", "search", search, "error", err)
		u.ShowErrorDialog("Something went wrong", err, w)
	}
	entry := widget.NewEntry()
	entry.PlaceHolder = "Type to start searching..."
	entry.ActionItem = iwidget.NewIconButton(theme.CancelIcon(), func() {
		entry.SetText("")
	})
	entry.OnChanged = func(search string) {
		if len(search) < 3 {
			results = results[:0]
			list.Refresh()
			return
		}
		go func() {
			var err error
			results, err = u.EveUniverseService().ListEntitiesByPartialName(context.Background(), search)
			if err != nil {
				showErrorDialog(search, err)
				return
			}
			list.Refresh()
		}()
		go func() {
			ctx := context.Background()
			missingIDs, err := u.CharacterService().AddEveEntitiesFromCharacterSearchESI(
				ctx,
				characterID,
				search,
			)
			if err != nil {
				showErrorDialog(search, err)
				return
			}
			if len(missingIDs) == 0 {
				return // no need to update when not changed
			}
			results, err = u.EveUniverseService().ListEntitiesByPartialName(ctx, search)
			if err != nil {
				showErrorDialog(search, err)
				return
			}
			list.Refresh()
		}()
	}
	c := container.NewBorder(
		container.NewBorder(
			widget.NewLabel("Add Recipient"),
			nil,
			nil,
			widget.NewButton("Cancel", func() {
				modal.Hide()
			}),
			entry,
		),
		nil,
		nil,
		nil,
		list,
	)
	modal = widget.NewModalPopUp(c, w.Canvas())
	_, s := w.Canvas().InteractiveArea()
	modal.Resize(fyne.NewSize(s.Width, s.Height))
	modal.Show()
	w.Canvas().Focus(entry)
}
