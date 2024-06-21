package chartbuilder_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/chartbuilder"
)

func TestCharts(t *testing.T) {
	t.Run("can create a bar chart", func(t *testing.T) {
		cb := chartbuilder.New()
		v := []chartbuilder.Value{
			{"Alpha", 2},
			{"Bravo", 3},
		}
		cb.Render(chartbuilder.Bar, "Title", v)
	})
	t.Run("can create a pie chart", func(t *testing.T) {
		cb := chartbuilder.New()
		v := []chartbuilder.Value{
			{"Alpha", 2},
			{"Bravo", 3},
		}
		cb.Render(chartbuilder.Pie, "Title", v)
	})
	t.Run("can handle no data", func(t *testing.T) {
		cb := chartbuilder.New()
		v := []chartbuilder.Value{}
		cb.Render(chartbuilder.Bar, "Title", v)
	})
	t.Run("can handle insufficient data", func(t *testing.T) {
		cb := chartbuilder.New()
		v := []chartbuilder.Value{
			{"Alpha", 0},
			{"Bravo", 0},
		}
		cb.Render(chartbuilder.Bar, "Title", v)
	})
}
