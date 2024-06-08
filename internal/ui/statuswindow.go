package ui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/dustin/go-humanize"
)

type statusCharacter struct {
	id         int32
	name       string
	completion float32
	isOK       bool
}

type statusWindow struct {
	characters        *widget.List
	charactersData    binding.UntypedList
	charactersTop     *widget.Label
	content           fyne.CanvasObject
	details           *fyne.Container
	sections          *widget.GridWrap
	characterSelected binding.Untyped
	sectionSelected   binding.Int
	sectionsData      binding.UntypedList
	sectionsTop       *widget.Label
	window            fyne.Window
	ui                *ui
}

func (u *ui) showStatusWindow() {
	if u.statusWindow != nil {
		u.statusWindow.Show()
		return
	}
	sw, err := u.newStatusWindow()
	if err != nil {
		panic(err)
	}
	if err := sw.refresh(); err != nil {
		panic(err)
	}
	w := u.app.NewWindow("Status")
	w.SetContent(sw.content)
	w.Resize(fyne.Size{Width: 1100, Height: 500})
	w.Show()
	ctx, cancel := context.WithCancel(context.TODO())
	sw.startTicker(ctx)
	w.SetOnClosed(func() {
		cancel()
		u.statusWindow = nil
	})
	u.statusWindow = w
	sw.window = w
}

func (u *ui) newStatusWindow() (*statusWindow, error) {
	a := &statusWindow{
		charactersData:    binding.NewUntypedList(),
		charactersTop:     widget.NewLabel(""),
		characterSelected: binding.NewUntyped(),
		sectionsData:      binding.NewUntypedList(),
		sectionsTop:       widget.NewLabel(""),
		sectionSelected:   binding.NewInt(),
		ui:                u,
	}

	if err := a.sectionSelected.Set(-1); err != nil {
		return nil, err
	}
	a.characters = a.makeCharacterList()
	a.charactersTop.TextStyle.Bold = true
	top1 := container.NewVBox(a.charactersTop, widget.NewSeparator())
	characters := container.NewBorder(top1, nil, nil, nil, a.characters)

	a.sections = a.makeSectionsTable()
	a.sectionsTop.TextStyle.Bold = true
	b := widget.NewButton("Force update all sections", func() {
		x, err := a.characterSelected.Get()
		if err != nil {
			panic(err)
		}
		c, ok := x.(statusCharacter)
		if !ok {
			return
		}
		a.ui.updateCharacterAndRefreshIfNeeded(context.Background(), c.id, false)
	})
	top2 := container.NewVBox(container.NewHBox(a.sectionsTop, layout.NewSpacer(), b), widget.NewSeparator())
	sections := container.NewBorder(top2, nil, nil, nil, a.sections)

	var vs *fyne.Container
	headline := widget.NewLabel("Section details")
	headline.TextStyle.Bold = true
	a.details = container.NewVBox()

	vs = container.NewBorder(nil, container.NewVBox(widget.NewSeparator(), a.details), nil, nil, sections)
	hs := container.NewHSplit(characters, vs)
	hs.SetOffset(0.33)
	a.content = hs
	return a, nil
}

func (a *statusWindow) makeCharacterList() *widget.List {
	list := widget.NewListWithData(
		a.charactersData,
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
			icon.FillMode = canvas.ImageFillContain
			icon.SetMinSize(fyne.Size{Width: defaultIconSize, Height: defaultIconSize})
			name := widget.NewLabel("Template")
			status := widget.NewLabel("Template")
			row := container.NewHBox(icon, name, layout.NewSpacer(), status)
			return row

			// hasToken, err := a.ui.service.HasTokenWithScopes(char.ID)
			// if err != nil {
			// 	slog.Error("Can not check if character has token", "err", err)
			// 	continue
			// }
			// if !hasToken {
			// 	row.Add(widget.NewIcon(theme.WarningIcon()))
			// }

		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			name := row.Objects[1].(*widget.Label)
			c, err := convertDataItem[statusCharacter](di)
			if err != nil {
				slog.Error("failed to render row account table", "err", err)
				name.Text = "failed to render"
				name.Importance = widget.DangerImportance
				name.Refresh()
				return
			}
			name.SetText(c.name)

			icon := row.Objects[0].(*canvas.Image)
			refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
				return a.ui.sv.EveImage.CharacterPortrait(c.id, defaultIconSize)
			})

			status := row.Objects[3].(*widget.Label)
			var t string
			var i widget.Importance
			if !c.isOK {
				t = "ERROR"
				i = widget.DangerImportance
			} else if c.completion < 1 {
				t = fmt.Sprintf("%.0f%% Fresh", c.completion*100)
				i = widget.HighImportance
			} else {
				t = "OK"
				i = widget.SuccessImportance
			}
			status.Text = t
			status.Importance = i
			status.Refresh()
		})

	list.OnSelected = func(id widget.ListItemID) {
		c, err := getItemUntypedList[statusCharacter](a.charactersData, id)
		if err != nil {
			slog.Error("failed to access account character in list", "err", err)
			list.UnselectAll()
			return
		}
		if err := a.characterSelected.Set(c); err != nil {
			panic(err)
		}
		if err := a.sectionSelected.Set(-1); err != nil {
			panic(err)
		}
		a.refreshDetailArea()
	}
	return list
}

func (a *statusWindow) makeSectionsTable() *widget.GridWrap {
	l := widget.NewGridWrapWithData(
		a.sectionsData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Section name long"),
				layout.NewSpacer(),
				widget.NewLabel("Status XXXX"),
			)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			cs, err := convertDataItem[model.CharacterStatus](di)
			if err != nil {
				panic(err)
			}
			row := co.(*fyne.Container)
			name := row.Objects[0].(*widget.Label)
			status := row.Objects[2].(*widget.Label)
			name.SetText(cs.Section.DisplayName())
			s, i := statusDisplay(cs)
			status.Text = s
			status.Importance = i
			status.Refresh()
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		if err := a.sectionSelected.Set(id); err != nil {
			slog.Error("Failed to select status entry", "err", err)
			l.UnselectAll()
			return
		}
		a.setDetails()
	}
	return l
}

type formItems struct {
	label      string
	value      string
	wrap       bool
	importance widget.Importance
}

type sectionStatusData struct {
	characterID   int32
	characterName string
	errorText     string
	sectionName   string
	lastUpdate    string
	section       model.CharacterSection
	sv            string
	si            widget.Importance
	timeout       string
}

func (a *statusWindow) setDetails() {
	var d sectionStatusData
	cs, found, err := a.fetchSelectedCharacterStatus()
	if err != nil {
		slog.Error("Failed to fetch selected character status")
	} else if found {
		d.characterID = cs.CharacterID
		d.section = cs.Section
		d.sv, d.si = statusDisplay(cs)
		if cs.ErrorMessage == "" {
			d.errorText = "-"
		} else {
			d.errorText = cs.ErrorMessage
		}
		d.characterName = cs.CharacterName
		d.sectionName = cs.Section.DisplayName()
		d.lastUpdate = lastUpdatedAtDisplay(cs)
		d.timeout = timeoutDisplay(cs)
	}
	oo := a.makeDetailsContent(d)
	a.details.RemoveAll()
	for _, o := range oo {
		a.details.Add(o)
	}
}

func (a *statusWindow) fetchSelectedCharacterStatus() (model.CharacterStatus, bool, error) {
	var z model.CharacterStatus
	id, err := a.sectionSelected.Get()
	if err != nil {
		return z, false, err
	}
	if id == -1 {
		return z, false, nil
	}
	cs, err := getItemUntypedList[model.CharacterStatus](a.sectionsData, id)
	if err != nil {
		return z, false, err
	}
	return cs, true, nil
}

func (a *statusWindow) makeDetailsContent(d sectionStatusData) []fyne.CanvasObject {
	items := []formItems{
		{label: "Section", value: d.sectionName},
		{label: "Status", value: d.sv, importance: d.si},
		{label: "Error", value: d.errorText, wrap: true},
		{label: "Last Update", value: d.lastUpdate},
		{label: "Timeout", value: d.timeout},
	}
	oo := make([]fyne.CanvasObject, 0)
	formLeading := makeForm(items[0:3])
	formTrailing := makeForm(items[3:])
	oo = append(oo, container.NewGridWithColumns(2, formLeading, formTrailing))
	oo = append(oo, widget.NewButton(fmt.Sprintf("Force update %s", d.sectionName), func() {
		if d.characterID != 0 {
			go a.ui.updateCharacterSectionAndRefreshIfNeeded(context.Background(), d.characterID, d.section, true)
		}

	}))
	return oo
}

func makeForm(data []formItems) *widget.Form {
	form := widget.NewForm()
	for _, row := range data {
		c := widget.NewLabel(row.value)
		if row.wrap {
			c.Wrapping = fyne.TextWrapWord
		}
		if row.importance != 0 {
			c.Importance = row.importance
		}
		form.Append(row.label, c)
	}
	return form
}

func (a *statusWindow) refresh() error {
	cc := a.ui.sv.CharacterStatus.ListCharacters()
	cc2 := make([]statusCharacter, len(cc))
	for i, c := range cc {
		completion, ok := a.ui.sv.CharacterStatus.CharacterSummary(c.ID)
		cc2[i] = statusCharacter{id: c.ID, name: c.Name, completion: completion, isOK: ok}
	}
	if err := a.charactersData.Set(copyToUntypedSlice(cc2)); err != nil {
		return err
	}
	a.charactersTop.SetText(fmt.Sprintf("Characters: %d", a.charactersData.Length()))
	a.characters.Refresh()
	a.refreshDetailArea()
	return nil
}

func (a *statusWindow) refreshDetailArea() error {
	x, err := a.characterSelected.Get()
	if err != nil {
		return err
	}
	c, ok := x.(statusCharacter)
	if !ok {
		return nil
	}
	data := a.ui.sv.CharacterStatus.ListStatus(c.id)
	if err := a.sectionsData.Set(copyToUntypedSlice(data)); err != nil {
		return err
	}
	a.sections.Refresh()
	a.sectionsTop.SetText(fmt.Sprintf("Update status for %s", c.name))
	a.setDetails()
	return nil
}

func (a *statusWindow) startTicker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				a.refresh()
				<-ticker.C
			}
		}
	}()
}

func lastUpdatedAtDisplay(cs model.CharacterStatus) string {
	return humanizeTime(cs.LastUpdatedAt, "?")
}

func timeoutDisplay(cs model.CharacterStatus) string {
	now := time.Now()
	return humanize.RelTime(now.Add(cs.Section.Timeout()), now, "", "")
}

func statusDisplay(cs model.CharacterStatus) (string, widget.Importance) {
	var s string
	var i widget.Importance
	if !cs.IsOK() {
		s = "ERROR"
		i = widget.DangerImportance
	} else if !cs.IsCurrent() {
		s = "Stale"
		i = widget.HighImportance
	} else {
		s = "OK"
		i = widget.SuccessImportance
	}
	return s, i
}
