package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestJobStatusDBValues_AreComplete(t *testing.T) {
	var count int
	for s := app.JobUndefined; s <= app.JobUnknown; s++ {
		v, ok := jobStatusToDBValue[s]
		require.True(t, ok, "missing job status: %s", s)
		xassert.Equal(t, s, jobStatusFromDBValue[v])
		count++
	}
	assert.Len(t, jobStatusFromDBValue, count)
}
