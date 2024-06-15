package ui

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/eveonline/icons"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service/statuscache"
	"github.com/dustin/go-humanize"
)

// An entity which has update sections, e.g. a character
type sectionEntity struct {
	id         int32
	name       string
	completion float32
	isOK       bool
}

type statusWindow struct {
	characters      *widget.List
	entitiesData    binding.UntypedList
	charactersTop   *widget.Label
	content         fyne.CanvasObject
	details         *fyne.Container
	sections        *widget.GridWrap
	entitySelected  binding.Untyped
	sectionSelected binding.Int
	sectionsData    binding.UntypedList
	sectionsTop     *widget.Label
	window          fyne.Window
	ui              *ui

	mu sync.Mutex
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
		entitiesData:    binding.NewUntypedList(),
		charactersTop:   widget.NewLabel(""),
		entitySelected:  binding.NewUntyped(),
		sectionsData:    binding.NewUntypedList(),
		sectionsTop:     widget.NewLabel(""),
		sectionSelected: binding.NewInt(),
		ui:              u,
	}

	if err := a.sectionSelected.Set(-1); err != nil {
		return nil, err
	}
	a.characters = a.makeEntityList()
	a.charactersTop.TextStyle.Bold = true
	top1 := container.NewVBox(a.charactersTop, widget.NewSeparator())
	characters := container.NewBorder(top1, nil, nil, nil, a.characters)

	a.sections = a.makeSectionTable()
	a.sectionsTop.TextStyle.Bold = true
	b := widget.NewButton("Force update all sections", func() {
		x, err := a.entitySelected.Get()
		if err != nil {
			panic(err)
		}
		c, ok := x.(sectionEntity)
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

func (a *statusWindow) makeEntityList() *widget.List {
	list := widget.NewListWithData(
		a.entitiesData,
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
			c, err := convertDataItem[sectionEntity](di)
			if err != nil {
				slog.Error("failed to render row account table", "err", err)
				name.Text = "failed to render"
				name.Importance = widget.DangerImportance
				name.Refresh()
				return
			}
			name.SetText(c.name)

			icon := row.Objects[0].(*canvas.Image)
			if c.id == statuscache.GeneralSectionID {
				icon.Resource = icons.GetResourceByName(icons.DataSheets)
				icon.Refresh()
			} else {
				refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
					return a.ui.sv.EveImage.CharacterPortrait(c.id, defaultIconSize)
				})
			}
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
		c, err := getItemUntypedList[sectionEntity](a.entitiesData, id)
		if err != nil {
			slog.Error("failed to access section entity in list", "err", err)
			list.UnselectAll()
			return
		}
		if err := a.entitySelected.Set(c); err != nil {
			panic(err)
		}
		if err := a.sectionSelected.Set(-1); err != nil {
			panic(err)
		}
		a.sections.UnselectAll()
		a.refreshDetailArea()
	}
	return list
}

func (a *statusWindow) makeSectionTable() *widget.GridWrap {
	l := widget.NewGridWrapWithData(
		a.sectionsData,
		func() fyne.CanvasObject {
			pb := widget.NewProgressBarInfinite()
			pb.Stop()
			return container.NewHBox(
				widget.NewLabel("Section name long"),
				pb,
				layout.NewSpacer(),
				widget.NewLabel("Status XXXX"),
			)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			cs, err := convertDataItem[model.SectionStatus](di)
			if err != nil {
				panic(err)
			}
			hbox := co.(*fyne.Container)
			name := hbox.Objects[0].(*widget.Label)
			status := hbox.Objects[3].(*widget.Label)
			name.SetText(cs.SectionName)
			s, i := statusDisplay(cs)
			status.Text = s
			status.Importance = i
			status.Refresh()

			pb := hbox.Objects[1].(*widget.ProgressBarInfinite)
			if cs.IsRunning() {
				pb.Start()
				pb.Show()
			} else {
				pb.Stop()
				pb.Hide()
			}
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
	entityID    int32
	entityName  string
	errorText   string
	sectionName string
	completedAt string
	startedAt   string
	sectionID   string
	sv          string
	si          widget.Importance
	timeout     string
}

func (a *statusWindow) setDetails() {
	var d sectionStatusData
	ss, found, err := a.fetchSelectedEntityStatus()
	if err != nil {
		slog.Error("Failed to fetch selected entity status")
	} else if found {
		d.entityID = ss.EntityID
		d.sectionID = ss.SectionID
		d.sv, d.si = statusDisplay(ss)
		if ss.ErrorMessage == "" {
			d.errorText = "-"
		} else {
			d.errorText = ss.ErrorMessage
		}
		d.entityName = ss.EntityName
		d.sectionName = ss.SectionName
		d.completedAt = humanizeTime(ss.CompletedAt, "?")
		d.startedAt = humanizeTime(ss.StartedAt, "-")
		now := time.Now()
		d.timeout = humanize.RelTime(now.Add(ss.Timeout), now, "", "")
	}
	oo := a.makeDetailsContent(d)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.details.RemoveAll()
	for _, o := range oo {
		a.details.Add(o)
	}
}

func (a *statusWindow) fetchSelectedEntityStatus() (model.SectionStatus, bool, error) {
	id, err := a.sectionSelected.Get()
	if err != nil {
		return model.SectionStatus{}, false, err
	}
	if id == -1 {
		return model.SectionStatus{}, false, nil
	}
	cs, err := getItemUntypedList[model.SectionStatus](a.sectionsData, id)
	if err != nil {
		return model.SectionStatus{}, false, err
	}
	return cs, true, nil
}

func (a *statusWindow) makeDetailsContent(d sectionStatusData) []fyne.CanvasObject {
	items := []formItems{
		{label: "Section", value: d.sectionName},
		{label: "Status", value: d.sv, importance: d.si},
		{label: "Error", value: d.errorText, wrap: true},
		{label: "Started", value: d.startedAt},
		{label: "Completed", value: d.completedAt},
		{label: "Timeout", value: d.timeout},
	}
	oo := make([]fyne.CanvasObject, 0)
	formLeading := makeForm(items[0:3])
	formTrailing := makeForm(items[3:])
	oo = append(oo, container.NewGridWithColumns(2, formLeading, formTrailing))
	oo = append(oo, widget.NewButton(fmt.Sprintf("Force update %s", d.sectionName), func() {
		if d.entityID != 0 {
			go a.ui.updateCharacterSectionAndRefreshIfNeeded(
				context.Background(), d.entityID, model.CharacterSection(d.sectionID), true)
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
	a.refreshEntityList()
	a.refreshDetailArea()
	a.charactersTop.SetText(fmt.Sprintf("Entities: %d", a.entitiesData.Length()))
	return nil
}

func (a *statusWindow) refreshEntityList() error {
	entities := make([]sectionEntity, 0)
	cc := a.ui.sv.StatusCache.ListCharacters()
	for _, c := range cc {
		completion, ok := a.ui.sv.StatusCache.CharacterSectionSummary(c.ID)
		o := sectionEntity{id: c.ID, name: c.Name, completion: completion, isOK: ok}
		entities = append(entities, o)
	}
	completion, ok := a.ui.sv.StatusCache.GeneralSectionSummary()
	o := sectionEntity{
		id:         statuscache.GeneralSectionID,
		name:       statuscache.GeneralSectionName,
		completion: completion,
		isOK:       ok,
	}
	entities = append(entities, o)
	if err := a.entitiesData.Set(copyToUntypedSlice(entities)); err != nil {
		return err
	}
	a.characters.Refresh()
	return nil
}

func (a *statusWindow) refreshDetailArea() error {
	x, err := a.entitySelected.Get()
	if err != nil {
		return err
	}
	c, ok := x.(sectionEntity)
	if !ok {
		return nil
	}
	data := a.ui.sv.StatusCache.SectionList(c.id)
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

func statusDisplay(cs model.SectionStatus) (string, widget.Importance) {
	var s string
	var i widget.Importance
	if !cs.IsOK() {
		s = "ERROR"
		i = widget.DangerImportance
	} else if cs.IsMissing() {
		s = "Missing"
		i = widget.WarningImportance
	} else if !cs.IsCurrent() {
		s = "Stale"
		i = widget.HighImportance
	} else {
		s = "OK"
		i = widget.SuccessImportance
	}
	return s, i
}
