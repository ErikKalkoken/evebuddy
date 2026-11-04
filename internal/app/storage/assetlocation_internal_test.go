package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestLocationFlagDBValues_AreComplete(t *testing.T) {
	var count int
	for f := app.FlagUndefined; f <= app.FlagUnknown; f++ {
		v, ok := locationFlagToDBValue[f]
		require.True(t, ok, "missing location flag: %s", f)
		xassert.Equal(t, f, locationFlagFromDBValue[v])
		count++
	}
	assert.Len(t, locationFlagFromDBValue, count)
}

func TestLocationTypeDBValues_AreComplete(t *testing.T) {
	var count int
	for x := app.TypeUndefined; x <= app.TypeUnknown; x++ {
		v, ok := locationTypeToDBValue[x]
		require.True(t, ok, "missing location type: %d", x)
		xassert.Equal(t, x, locationTypeFromDBValue[v])
		count++
	}
	assert.Len(t, locationTypeFromDBValue, count)
}
