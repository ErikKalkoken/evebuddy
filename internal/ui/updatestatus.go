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
	detailsInner      *fyne.Container
	details           *fyne.Container
	sections          *widget.Table
	characterSelected binding.Untyped
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
	w.Resize(fyne.Size{Width: 1000, Height: 500})
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
		ui:                u,
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
		a.ui.updateCharacterAndRefreshIfNeeded(c.id, false)
	})
	top2 := container.NewVBox(container.NewHBox(a.sectionsTop, layout.NewSpacer(), b), widget.NewSeparator())
	sections := container.NewBorder(top2, nil, nil, nil, a.sections)

	a.detailsInner = container.NewVBox()
	headline := widget.NewLabel("Section details")
	headline.TextStyle.Bold = true
	top3 := container.NewVBox(headline, widget.NewSeparator())
	a.details = container.NewBorder(top3, nil, nil, nil, a.detailsInner)
	a.details.Hide()

	vs := container.NewBorder(nil, a.details, nil, nil, sections)
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
				return a.ui.imageManager.CharacterPortrait(c.id, defaultIconSize)
			})

			status := row.Objects[3].(*widget.Label)
			var t string
			var i widget.Importance
			if !c.isOK {
				t = "ERROR"
				i = widget.DangerImportance
			} else if c.completion < 1 {
				t = fmt.Sprintf("Updating %.0f%%...", c.completion*100)
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
			return
		}
		if err := a.characterSelected.Set(c); err != nil {
			panic(err)
		}
	}
	return list
}

func (a *statusWindow) makeSectionsTable() *widget.Table {
	var headers = []struct {
		text  string
		width float32
	}{
		{"Section", 150},
		{"Timeout", 150},
		{"Last Update", 150},
		{"Status", 150},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return a.sectionsData.Length(), len(headers)

		},
		func() fyne.CanvasObject {
			l := widget.NewLabel("Placeholder")
			l.Truncation = fyne.TextTruncateEllipsis
			return l
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			cs, err := getItemUntypedList[model.CharacterStatus](a.sectionsData, tci.Row)
			if err != nil {
				panic(err)
			}
			label := co.(*widget.Label)
			var s string
			i := widget.MediumImportance
			switch tci.Col {
			case 0:
				s = cs.Section.Name()
			case 1:
				s = timeoutOutput(cs)
			case 2:
				s = lastUpdatedAtOutput(cs)
			case 3:
				s, i = statusOutput(cs)
			}
			label.Text = s
			label.Importance = i
			label.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		co.(*widget.Label).SetText(s.text)
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.width)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		cs, err := getItemUntypedList[model.CharacterStatus](a.sectionsData, tci.Row)
		if err != nil {
			t := "Failed to select status entry"
			slog.Error(t, "err", err)
			a.ui.statusBarArea.SetError(t)
			return
		}
		a.setDetails(cs)
	}
	return t
}

func statusOutput(cs model.CharacterStatus) (string, widget.Importance) {
	var s string
	var i widget.Importance
	if !cs.IsCurrent() {
		s = "Stale"
		i = widget.HighImportance
	} else if cs.IsOK() {
		s = "OK"
		i = widget.SuccessImportance
	} else {
		s = "ERROR"
		i = widget.DangerImportance
	}
	return s, i
}

type formItems struct {
	label      string
	value      string
	wrap       bool
	importance widget.Importance
}

func (a *statusWindow) setDetails(cs model.CharacterStatus) {
	sv, si := statusOutput(cs)
	var errorText string
	if cs.ErrorMessage == "" {
		errorText = "-"
	} else {
		errorText = cs.ErrorMessage
	}
	leading := []formItems{
		{label: "Character", value: cs.CharacterName},
		{label: "Section", value: cs.Section.Name()},
		{label: "Status", value: sv, importance: si},
		{label: "Error", value: errorText, wrap: true},
	}
	formLeading := makeForm(leading)
	trailing := []formItems{
		{label: "Last Update", value: lastUpdatedAtOutput(cs)},
		{label: "Timeout", value: timeoutOutput(cs)},
	}
	formTrailing := makeForm(trailing)
	a.detailsInner.RemoveAll()
	a.detailsInner.Add(container.NewGridWithColumns(2, formLeading, formTrailing))
	bUpdate := widget.NewButton("Force update", func() {
		go a.ui.updateCharacterSectionAndRefreshIfNeeded(cs.CharacterID, cs.Section, true)
	})
	a.detailsInner.Add(bUpdate)
	a.details.Show()
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
	cc := a.ui.service.CharacterStatusListCharacters()
	cc2 := make([]statusCharacter, len(cc))
	for i, c := range cc {
		completed, ok := a.ui.service.CharacterStatusCharacterSummary(c.ID)
		cc2[i] = statusCharacter{id: c.ID, name: c.Name, completion: completed, isOK: ok}
	}
	if err := a.charactersData.Set(copyToUntypedSlice(cc2)); err != nil {
		return err
	}
	a.charactersTop.SetText(fmt.Sprintf("Characters: %d", a.charactersData.Length()))
	a.characters.Refresh()
	a.refreshDetailArea()
	a.sections.Refresh()
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
	data := a.ui.service.CharacterStatusListStatus(c.id)
	if err := a.sectionsData.Set(copyToUntypedSlice(data)); err != nil {
		return err
	}
	a.sectionsTop.SetText(fmt.Sprintf("Update status for %s", c.name))
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

func lastUpdatedAtOutput(cs model.CharacterStatus) string {
	return humanizeTime(cs.LastUpdatedAt, "?")
}

func timeoutOutput(cs model.CharacterStatus) string {
	now := time.Now()
	return humanize.RelTime(now.Add(cs.Section.Timeout()), now, "", "")
}
