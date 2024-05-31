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
	characters     *widget.List
	charactersData binding.UntypedList
	charactersTop  *widget.Label
	content        fyne.CanvasObject
	detail         *widget.Table
	detailSelected binding.Untyped
	detailData     binding.UntypedList
	detailTop      *widget.Label
	ui             *ui
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
	w.Resize(fyne.Size{Width: 800, Height: 500})
	w.Show()
	ctx, cancel := context.WithCancel(context.TODO())
	sw.startTicker(ctx)
	w.SetOnClosed(func() {
		cancel()
		u.statusWindow = nil
	})
	u.statusWindow = w
}

func (u *ui) newStatusWindow() (*statusWindow, error) {
	a := &statusWindow{
		charactersData: binding.NewUntypedList(),
		charactersTop:  widget.NewLabel(""),
		detailData:     binding.NewUntypedList(),
		detailSelected: binding.NewUntyped(),
		detailTop:      widget.NewLabel(""),
		ui:             u,
	}

	a.characters = a.makeCharacterList()
	a.charactersTop.TextStyle.Bold = true
	characters := container.NewBorder(a.charactersTop, nil, nil, nil, a.characters)

	a.detail = a.makeDetailTable()
	a.detailTop.TextStyle.Bold = true
	detail := container.NewBorder(a.detailTop, nil, nil, nil, a.detail)

	a.detailSelected.AddListener(binding.NewDataListener(func() {
		if err := a.refreshDetailArea(); err != nil {
			panic(err)
		}
	}))

	split := container.NewHSplit(characters, detail)
	split.SetOffset(0.33)
	a.content = split
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
		if err := a.detailSelected.Set(c); err != nil {
			panic(err)
		}
	}
	return list
}

func (a *statusWindow) makeDetailTable() *widget.Table {
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
			return a.detailData.Length(), len(headers)

		},
		func() fyne.CanvasObject {
			l := widget.NewLabel("Placeholder")
			l.Truncation = fyne.TextTruncateEllipsis
			return l
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			d, err := getItemUntypedList[model.CharacterStatus](a.detailData, tci.Row)
			if err != nil {
				panic(err)
			}
			label := co.(*widget.Label)
			var s string
			i := widget.MediumImportance
			switch tci.Col {
			case 0:
				s = d.Section
			case 1:
				now := time.Now()
				s = humanize.RelTime(now.Add(d.Timeout), now, "", "")
			case 2:
				s = humanizeTime(d.LastUpdatedAt, "?")
			case 3:
				if d.IsOK() {
					s = "OK"
					i = widget.SuccessImportance
				} else {
					s = d.ErrorMessage
					i = widget.DangerImportance
				}
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
	t.OnSelected = func(id widget.TableCellID) {
		t.UnselectAll()
	}
	return t
}

func (a *statusWindow) refresh() error {
	cc, err := a.ui.service.ListCharactersShort()
	if err != nil {
		return err
	}
	cc2 := make([]statusCharacter, len(cc))
	for i, c := range cc {
		completed, ok := a.ui.service.CharacterGetUpdateStatusCharacterSummary(c.ID)
		cc2[i] = statusCharacter{id: c.ID, name: c.Name, completion: completed, isOK: ok}
	}
	if err := a.charactersData.Set(copyToUntypedSlice(cc2)); err != nil {
		return err
	}
	a.charactersTop.SetText(fmt.Sprintf("Characters: %d", a.charactersData.Length()))
	a.characters.Refresh()
	a.refreshDetailArea()
	a.detail.Refresh()
	return nil
}

func (a *statusWindow) refreshDetailArea() error {
	x, err := a.detailSelected.Get()
	if err != nil {
		return err
	}
	c, ok := x.(statusCharacter)
	if !ok {
		return nil
	}
	data := a.ui.service.CharacterListUpdateStatus(c.id)
	if err := a.detailData.Set(copyToUntypedSlice(data)); err != nil {
		return err
	}
	a.detailTop.SetText(fmt.Sprintf("Update status for %s", c.name))
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
