package chartbuilder_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/evebuddy/internal/app/chartbuilder"
)

func TestCharts(t *testing.T) {
	size := fyne.NewSize(300, 300)
	t.Run("can create a bar chart", func(t *testing.T) {
		cb := chartbuilder.New(nil)
		v := []chartbuilder.Value{
			{"Alpha", 2},
			{"Bravo", 3},
		}
		cb.Render(chartbuilder.Bar, size, "Title", v)
	})
	t.Run("can create a pie chart", func(t *testing.T) {
		cb := chartbuilder.New(nil)
		v := []chartbuilder.Value{
			{"Alpha", 2},
			{"Bravo", 3},
		}
		cb.Render(chartbuilder.Pie, size, "Title", v)
	})
	t.Run("can handle no data", func(t *testing.T) {
		cb := chartbuilder.New(nil)
		v := []chartbuilder.Value{}
		cb.Render(chartbuilder.Bar, size, "Title", v)
	})
	t.Run("can handle insufficient data", func(t *testing.T) {
		cb := chartbuilder.New(nil)
		v := []chartbuilder.Value{
			{"Alpha", 0},
			{"Bravo", 0},
		}
		cb.Render(chartbuilder.Bar, size, "Title", v)
	})
}
