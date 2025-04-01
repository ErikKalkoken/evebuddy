package allcharacters

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type trainingCharacter struct {
	id            int32
	name          string
	totalSP       optional.Optional[int]
	training      optional.Optional[time.Duration]
	unallocatedSP optional.Optional[int]
}

type TrainingOverview struct {
	widget.BaseWidget

	body fyne.CanvasObject
	rows []trainingCharacter
	top  *widget.Label
	u    app.UI
}

func NewTrainingOverview(u app.UI) *TrainingOverview {
	a := &TrainingOverview{
		rows: make([]trainingCharacter, 0),
		top:  appwidget.MakeTopLabel(),
		u:    u,
	}
	a.ExtendBaseWidget(a)
	headers := []iwidget.HeaderDef{
		{Text: "Name", Width: 250},
		{Text: "SP", Width: 100},
		{Text: "Unall. SP", Width: 100},
		{Text: "Training", Width: 100},
	}
	makeDataLabel := func(col int, c trainingCharacter) (string, fyne.TextAlign, widget.Importance) {
		var align fyne.TextAlign
		var importance widget.Importance
		var text string
		switch col {
		case 0:
			text = c.name
		case 1:
			text = ihumanize.Optional(c.totalSP, "?")
			align = fyne.TextAlignTrailing
		case 2:
			text = ihumanize.Optional(c.unallocatedSP, "?")
			align = fyne.TextAlignTrailing
		case 3:
			if c.training.IsEmpty() {
				text = "Inactive"
				importance = widget.WarningImportance
			} else {
				text = ihumanize.Duration(c.training.ValueOrZero())
			}
		}
		return text, align, importance
	}
	if a.u.IsDesktop() {
		a.body = iwidget.MakeDataTableForDesktop(headers, &a.rows, makeDataLabel, nil)
	} else {
		a.body = iwidget.MakeDataTableForMobile(headers, &a.rows, makeDataLabel, nil)
	}
	return a
}

func (a *TrainingOverview) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *TrainingOverview) Update() {
	t, i, err := func() (string, widget.Importance, error) {
		totalSP, err := a.updateCharacters()
		if err != nil {
			return "", 0, err
		}
		if len(a.rows) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		spText := ihumanize.Optional(totalSP, "?")
		s := fmt.Sprintf("%d characters • %s Total SP", len(a.rows), spText)
		return s, widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh overview UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.body.Refresh()
}

func (a *TrainingOverview) updateCharacters() (optional.Optional[int], error) {
	var totalSP optional.Optional[int]
	var err error
	ctx := context.TODO()
	mycc, err := a.u.CharacterService().ListCharacters(ctx)
	if err != nil {
		return totalSP, err
	}
	cc := make([]trainingCharacter, len(mycc))
	for i, m := range mycc {
		c := trainingCharacter{
			id:            m.ID,
			name:          m.EveCharacter.Name,
			totalSP:       m.TotalSP,
			unallocatedSP: m.UnallocatedSP,
		}
		cc[i] = c
	}
	for i, c := range cc {
		v, err := a.u.CharacterService().GetCharacterTotalTrainingTime(ctx, c.id)
		if err != nil {
			return totalSP, err
		}
		cc[i].training = v
	}
	for _, c := range cc {
		if !c.totalSP.IsEmpty() {
			totalSP.Set(totalSP.ValueOrZero() + c.totalSP.ValueOrZero())
		}
	}
	a.rows = cc
	return totalSP, nil
}
